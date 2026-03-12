package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis-developer/redis-starter-go/cmd/config"
	"github.com/redis-developer/redis-starter-go/internal/testredis"
	server "github.com/redis-developer/redis-starter-go/pkg"
	"github.com/redis-developer/redis-starter-go/pkg/components/todos"
	"github.com/redis-developer/redis-starter-go/pkg/logging"
	redispkg "github.com/redis-developer/redis-starter-go/pkg/redis"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testApp struct {
	baseURL string
	client  *http.Client
}

type todo struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	CreatedDate string `json:"created_date"`
	UpdatedDate string `json:"updated_date"`
}

type todoDocument struct {
	ID    string `json:"id"`
	Value todo   `json:"value"`
}

type todosResponse struct {
	Total     int64          `json:"total"`
	Documents []todoDocument `json:"documents"`
}

type errorResponse struct {
	Message string `json:"message"`
}

func newTestApp(t *testing.T, logLevel string) (*testApp, *goredis.Client) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping Docker-backed integration test in short mode")
	}

	harness := testredis.Start(t)
	ctx := context.Background()

	client, err := redispkg.NewClient(ctx, harness.URL)
	require.NoError(t, err)

	logger, closeLogger, err := logging.New(logging.Options{
		Environment: "test",
		Level:       logLevel,
		StreamKey:   "logs",
		RedisClient: client,
		Output:      io.Discard,
	})
	require.NoError(t, err)

	model := todos.NewModel(client)
	require.NoError(t, model.CreateIndexIfNotExists(ctx))
	require.NoError(t, model.DelAll(ctx))

	srv := server.New(
		config.Config{
			AppEnv:       config.Test,
			Port:         8080,
			RedisURL:     harness.URL,
			LogLevel:     logLevel,
			LogStreamKey: "logs",
		},
		logger,
		model,
		closeLogger,
		func(context.Context) error {
			return client.Close()
		},
	)

	httpServer := httptest.NewServer(srv.Handler())
	t.Cleanup(func() {
		httpServer.Close()
		_ = model.DelAll(context.Background())
		model.DropIndex(context.Background())
		_ = closeLogger(context.Background())
		_ = client.Close()
	})

	return &testApp{
		baseURL: httpServer.URL,
		client:  httpServer.Client(),
	}, client
}

func TestTodosHTTPIntegration(t *testing.T) {
	app, _ := newTestApp(t, "silent")

	createResponse := todoDocument{}
	postJSON(t, app, "/api/todos", map[string]string{"name": "Take out the trash"}, http.StatusOK, &createResponse)
	assert.Equal(t, "Take out the trash", createResponse.Value.Name)
	assert.Equal(t, "todo", createResponse.Value.Status)

	readResponse := todo{}
	getJSON(t, app, "/api/todos/"+createResponse.ID, http.StatusOK, &readResponse)
	assert.Equal(t, createResponse.Value.Name, readResponse.Name)

	updateResponse := todo{}
	patchJSON(t, app, "/api/todos/"+createResponse.ID, map[string]string{"status": "complete"}, http.StatusOK, &updateResponse)
	assert.Equal(t, "complete", updateResponse.Status)
	assert.True(t, updateResponse.UpdatedDate > updateResponse.CreatedDate)

	allResponse := todosResponse{}
	getJSON(t, app, "/api/todos", http.StatusOK, &allResponse)
	assert.Equal(t, int64(1), allResponse.Total)

	deleteRequest, err := http.NewRequest(http.MethodDelete, app.baseURL+"/api/todos/"+createResponse.ID, nil)
	require.NoError(t, err)
	deleteResult, err := app.client.Do(deleteRequest)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = deleteResult.Body.Close()
	})
	body, err := io.ReadAll(deleteResult.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, deleteResult.StatusCode)
	assert.Equal(t, "ok", string(body))
}

func TestTodosSearchAndErrors(t *testing.T) {
	app, _ := newTestApp(t, "silent")

	first := todoDocument{}
	postJSON(t, app, "/api/todos", map[string]string{"name": "Buy groceries"}, http.StatusOK, &first)
	patchJSON(t, app, "/api/todos/"+first.ID, map[string]string{"status": "in progress"}, http.StatusOK, &todo{})

	second := todoDocument{}
	postJSON(t, app, "/api/todos", map[string]string{"name": "Walk dog"}, http.StatusOK, &second)

	searchByName := todosResponse{}
	getJSON(t, app, "/api/todos/search?name=Buy", http.StatusOK, &searchByName)
	assert.Equal(t, int64(1), searchByName.Total)
	assert.Equal(t, "Buy groceries", searchByName.Documents[0].Value.Name)

	searchByStatus := todosResponse{}
	getJSON(t, app, "/api/todos/search?status=in%20progress", http.StatusOK, &searchByStatus)
	assert.Equal(t, int64(1), searchByStatus.Total)
	assert.Equal(t, "Buy groceries", searchByStatus.Documents[0].Value.Name)

	searchNoFilters := todosResponse{}
	getJSON(t, app, "/api/todos/search", http.StatusOK, &searchNoFilters)
	assert.Equal(t, int64(2), searchNoFilters.Total)

	createErr := errorResponse{}
	postJSON(t, app, "/api/todos", map[string]string{"name": ""}, http.StatusBadRequest, &createErr)
	assert.Equal(t, "todo must have a name", createErr.Message)

	updateErr := errorResponse{}
	patchJSON(t, app, "/api/todos/"+first.ID, map[string]string{"status": "done"}, http.StatusBadRequest, &updateErr)
	assert.Equal(t, "invalid status \"done\"", updateErr.Message)

	notFoundErr := errorResponse{}
	getJSON(t, app, "/api/todos/missing", http.StatusNotFound, &notFoundErr)
	assert.Equal(t, "todo not found", notFoundErr.Message)
}

func TestRequestsAreWrittenToRedisLogStream(t *testing.T) {
	app, redisClient := newTestApp(t, "info")

	postJSON(t, app, "/api/todos", map[string]string{"name": "Write tests"}, http.StatusOK, &todoDocument{})

	assert.Eventually(t, func() bool {
		results, err := redisClient.XRevRange(context.Background(), "logs", "+", "-").Result()
		if err != nil || len(results) == 0 {
			return false
		}

		for _, result := range results {
			if result.Values["component"] == "http" && result.Values["msg"] == "request completed" {
				return true
			}
		}

		return false
	}, 5*time.Second, 100*time.Millisecond)
}

func getJSON(t *testing.T, app *testApp, path string, expectedStatus int, target any) {
	t.Helper()

	response, err := app.client.Get(app.baseURL + path)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = response.Body.Close()
	})

	require.Equal(t, expectedStatus, response.StatusCode)
	require.NoError(t, json.NewDecoder(response.Body).Decode(target))
}

func postJSON(t *testing.T, app *testApp, path string, body any, expectedStatus int, target any) {
	t.Helper()
	requestJSON(t, app, http.MethodPost, path, body, expectedStatus, target)
}

func patchJSON(t *testing.T, app *testApp, path string, body any, expectedStatus int, target any) {
	t.Helper()
	requestJSON(t, app, http.MethodPatch, path, body, expectedStatus, target)
}

func requestJSON(t *testing.T, app *testApp, method string, path string, body any, expectedStatus int, target any) {
	t.Helper()

	payload, err := json.Marshal(body)
	require.NoError(t, err)

	request, err := http.NewRequest(method, app.baseURL+path, bytes.NewReader(payload))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")

	response, err := app.client.Do(request)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = response.Body.Close()
	})

	require.Equal(t, expectedStatus, response.StatusCode)
	require.NoError(t, json.NewDecoder(response.Body).Decode(target))
}
