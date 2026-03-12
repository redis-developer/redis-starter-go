package todos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

// The redis index name for Todos.
const TodoIndex = "todos-idx"

// The key prefix for Todos stored in redis.
const TodoPrefix = "todos:"

// TodoStatus enumerates all the status types a Todo can have.
type TodoStatus string

// A Todo has one of these status values.
const (
	NotStarted TodoStatus = "todo"
	InProgress TodoStatus = "in progress"
	Complete   TodoStatus = "complete"
)

// TodoStatusMap maps a string to its corresponding TodoStatus.
var TodoStatusMap = map[string]TodoStatus{
	"todo":        NotStarted,
	"in progress": InProgress,
	"complete":    Complete,
}

// Todo is the base model used to store the name and status of Todos.
type Todo struct {
	Name   string     `json:"name"`
	Status TodoStatus `json:"status"`

	CreatedDate time.Time `json:"created_date"`
	UpdatedDate time.Time `json:"updated_date"`
}

// TodoDocument contains the Todo's ID in Redis as well as the Todo value.
type TodoDocument struct {
	ID    string `json:"id"`
	Value Todo   `json:"value"`
}

// Todos is the result of list-based operations against Redis.
type Todos struct {
	Total     int64          `json:"total"`
	Documents []TodoDocument `json:"documents"`
}

// Model provides the methods necessary to handle CRUD operations for Todos.
// It also handles creating the index for searching Todos.
type Model interface {
	CreateIndexIfNotExists(ctx context.Context) error
	DropIndex(ctx context.Context)
	All(ctx context.Context) (*Todos, error)
	One(ctx context.Context, id string) (*Todo, error)
	Search(ctx context.Context, name string, status string) (*Todos, error)
	Create(ctx context.Context, id string, name string) (*TodoDocument, error)
	Update(ctx context.Context, id string, status string) (*Todo, error)
	Del(ctx context.Context, id string) error
	DelAll(ctx context.Context) error
}

// TodoModel implements a Model with a provided Redis client.
type TodoModel struct {
	db *redis.Client
}

// TodoErrorType enumerates the types of internal errors thrown.
type TodoErrorType string

// A TodoError has one of these types.
const (
	NotFound TodoErrorType = "not found"
	Invalid  TodoErrorType = "invalid"
	Unknown  TodoErrorType = "unknown"
)

// TodoError is a Redis-level error with a safe message for the client.
type TodoError struct {
	ErrorType     TodoErrorType
	ClientMessage string
	Err           error
}

// Error makes a TodoError operate as a Go error.
func (e *TodoError) Error() string {
	if e.Err == nil {
		return e.ClientMessage
	}

	return e.Err.Error()
}

func parseTodoStr(todoJSON string) (Todo, error) {
	todo := Todo{}
	if err := json.Unmarshal([]byte(todoJSON), &todo); err != nil {
		return Todo{}, err
	}

	return todo, nil
}

// formatId returns a normalized ID string that allows IDs to either
// include or exclude the TodoPrefix.
func formatId(id string) string {
	matched, err := regexp.Match(`^`+TodoPrefix, []byte(id))
	if err != nil {
		panic(err)
	}

	if matched {
		return id
	}

	return fmt.Sprintf("%s%s", TodoPrefix, id)
}

// haveIndex returns whether or not the Todo index already exists in Redis.
func (m TodoModel) haveIndex(ctx context.Context) bool {
	indexes := m.db.FT_List(ctx)
	return slices.Contains(indexes.Val(), TodoIndex)
}

// CreateIndexIfNotExists ensures that the Todos index exists in Redis.
func (m TodoModel) CreateIndexIfNotExists(ctx context.Context) error {
	if m.haveIndex(ctx) {
		return nil
	}

	_, err := m.db.FTCreate(
		ctx,
		TodoIndex,
		&redis.FTCreateOptions{
			OnJSON: true,
			Prefix: []any{TodoPrefix},
		},
		&redis.FieldSchema{
			FieldName: "$.name",
			As:        "name",
			FieldType: redis.SearchFieldTypeText,
		},
		&redis.FieldSchema{
			FieldName: "$.status",
			As:        "status",
			FieldType: redis.SearchFieldTypeTag,
		},
	).Result()
	if err != nil {
		return &TodoError{
			ErrorType:     Unknown,
			ClientMessage: "failed to create index",
			Err:           fmt.Errorf("failed to create index: %w", err),
		}
	}

	return nil
}

// DropIndex deletes the Todos index in Redis.
func (m TodoModel) DropIndex(ctx context.Context) {
	if !m.haveIndex(ctx) {
		return
	}

	_ = m.db.FTDropIndex(ctx, TodoIndex).Err()
}

// All returns all existing Todos in Redis.
func (m TodoModel) All(ctx context.Context) (*Todos, error) {
	todosResult, err := m.db.FTSearch(ctx, TodoIndex, "*").Result()
	if err != nil {
		return nil, &TodoError{
			ErrorType:     Unknown,
			ClientMessage: "failed to get all todos",
			Err:           fmt.Errorf("failed to get all todos: %w", err),
		}
	}

	documents, err := parseDocuments(todosResult.Docs)
	if err != nil {
		return nil, &TodoError{
			ErrorType:     Unknown,
			ClientMessage: "failed to decode todos",
			Err:           fmt.Errorf("failed to decode todos: %w", err),
		}
	}

	return &Todos{
		Total:     int64(todosResult.Total),
		Documents: documents,
	}, nil
}

// One returns a Todo if it exists in Redis based on the input id.
func (m TodoModel) One(ctx context.Context, id string) (*Todo, error) {
	todoStr, err := m.db.JSONGet(ctx, formatId(id)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, &TodoError{
				ErrorType:     NotFound,
				ClientMessage: "todo not found",
			}
		}

		return nil, &TodoError{
			ErrorType:     Unknown,
			ClientMessage: "failed to get todo",
			Err:           fmt.Errorf("failed to get todo: %w", err),
		}
	}

	if todoStr == "" {
		return nil, &TodoError{
			ErrorType:     NotFound,
			ClientMessage: "todo not found",
		}
	}

	todo, err := parseTodoStr(todoStr)
	if err != nil {
		return nil, &TodoError{
			ErrorType:     Unknown,
			ClientMessage: "failed to decode todo",
			Err:           fmt.Errorf("failed to decode todo: %w", err),
		}
	}

	return &todo, nil
}

// Search returns a Todos object with a list of Todos that match the
// input parameters (name and/or status).
func (m TodoModel) Search(ctx context.Context, name string, status string) (*Todos, error) {
	name = strings.TrimSpace(name)
	status = strings.TrimSpace(status)

	if name == "" && status == "" {
		return m.All(ctx)
	}

	var searches []string
	if name != "" {
		searches = append(searches, fmt.Sprintf("@name:(%s)", name))
	}
	if status != "" {
		searches = append(searches, fmt.Sprintf("@status:{%s}", strings.ReplaceAll(status, " ", `\ `)))
	}

	todosResult, err := m.db.FTSearchWithArgs(
		ctx,
		TodoIndex,
		strings.Join(searches, " "),
		&redis.FTSearchOptions{DialectVersion: 2},
	).Result()
	if err != nil {
		return nil, &TodoError{
			ErrorType:     Unknown,
			ClientMessage: "failed search for todos",
			Err:           fmt.Errorf("failed to search for todos: %w", err),
		}
	}

	documents, err := parseDocuments(todosResult.Docs)
	if err != nil {
		return nil, &TodoError{
			ErrorType:     Unknown,
			ClientMessage: "failed to decode todos",
			Err:           fmt.Errorf("failed to decode todos: %w", err),
		}
	}

	return &Todos{
		Total:     int64(todosResult.Total),
		Documents: documents,
	}, nil
}

// Create returns a newly created Todo from Redis based on the input ID and name.
func (m TodoModel) Create(ctx context.Context, id string, name string) (*TodoDocument, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, &TodoError{
			ErrorType:     Invalid,
			ClientMessage: "todo must have a name",
		}
	}

	if strings.TrimSpace(id) == "" {
		id = uuid.New().String()
	}

	now := time.Now().UTC()
	formattedID := formatId(strings.TrimSpace(id))
	todo := &TodoDocument{
		ID: formattedID,
		Value: Todo{
			Name:        name,
			Status:      NotStarted,
			CreatedDate: now,
			UpdatedDate: now,
		},
	}

	if _, err := m.db.JSONSet(ctx, formattedID, "$", todo.Value).Result(); err != nil {
		return nil, &TodoError{
			ErrorType:     Invalid,
			ClientMessage: "failed to create todo",
			Err:           err,
		}
	}

	return todo, nil
}

// Update returns an updated Todo from Redis based on the ID with a new status.
func (m TodoModel) Update(ctx context.Context, id string, status string) (*Todo, error) {
	todoStatus, ok := TodoStatusMap[status]
	if !ok {
		return nil, &TodoError{
			ErrorType:     Invalid,
			ClientMessage: fmt.Sprintf("invalid status %q", status),
		}
	}

	todo, err := m.One(ctx, id)
	if err != nil {
		return nil, err
	}

	todo.Status = todoStatus
	todo.UpdatedDate = time.Now().UTC()

	if _, err := m.db.JSONSet(ctx, formatId(id), "$", todo).Result(); err != nil {
		return nil, &TodoError{
			ErrorType:     Unknown,
			ClientMessage: "failed to update todo",
			Err:           fmt.Errorf("failed to update todo: %w", err),
		}
	}

	return todo, nil
}

// Del deletes a todo in Redis if it exists.
func (m TodoModel) Del(ctx context.Context, id string) error {
	if _, err := m.db.JSONDel(ctx, formatId(id), "$").Result(); err != nil {
		return &TodoError{
			ErrorType:     Unknown,
			ClientMessage: "failed to delete todo",
			Err:           fmt.Errorf("failed to delete todo: %w", err),
		}
	}

	return nil
}

// DelAll deletes all the Todos in Redis.
func (m TodoModel) DelAll(ctx context.Context) error {
	allTodos, err := m.All(ctx)
	if err != nil {
		return &TodoError{
			ErrorType:     Unknown,
			ClientMessage: "failed to find all todos",
			Err:           fmt.Errorf("failed to find all todos: %w", err),
		}
	}

	for _, todo := range allTodos.Documents {
		if err := m.Del(ctx, todo.ID); err != nil {
			return &TodoError{
				ErrorType:     Unknown,
				ClientMessage: "failed to delete todo",
				Err:           fmt.Errorf("failed to delete todo: %w", err),
			}
		}
	}

	return nil
}

func parseDocuments(docs []redis.Document) ([]TodoDocument, error) {
	documents := make([]TodoDocument, 0, len(docs))
	for _, todoDoc := range docs {
		todo, err := parseTodoStr(todoDoc.Fields["$"])
		if err != nil {
			return nil, err
		}

		documents = append(documents, TodoDocument{
			ID:    todoDoc.ID,
			Value: todo,
		})
	}

	return documents, nil
}

// NewModel returns a Model that uses the passed-in Redis client
// to manage Todos.
func NewModel(db *redis.Client) *TodoModel {
	return &TodoModel{db: db}
}
