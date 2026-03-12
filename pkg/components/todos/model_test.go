package todos

import (
	"context"
	"testing"

	"github.com/redis-developer/redis-starter-go/internal/testredis"
	redispkg "github.com/redis-developer/redis-starter-go/pkg/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestModel(t *testing.T) (*TodoModel, context.Context) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping Redis integration test in short mode")
	}

	harness := testredis.Start(t)
	ctx := context.Background()

	client, err := redispkg.NewClient(ctx, harness.URL)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = client.Close()
	})

	model := NewModel(client)
	require.NoError(t, model.CreateIndexIfNotExists(ctx))
	require.NoError(t, model.DelAll(ctx))

	t.Cleanup(func() {
		_ = model.DelAll(ctx)
		model.DropIndex(ctx)
	})

	return model, ctx
}

func TestCrud(t *testing.T) {
	model, ctx := newTestModel(t)

	t.Run("CRUD for a single todo", func(t *testing.T) {
		sampleTodo := &TodoDocument{
			ID: "todos:abc123",
			Value: Todo{
				Name:   "Take out the trash",
				Status: NotStarted,
			},
		}

		todo, err := model.Create(ctx, sampleTodo.ID, sampleTodo.Value.Name)
		require.NoError(t, err)
		assert.Equal(t, sampleTodo.ID, todo.ID)
		assert.Equal(t, sampleTodo.Value.Name, todo.Value.Name)
		assert.Equal(t, sampleTodo.Value.Status, todo.Value.Status)

		readResult, err := model.One(ctx, todo.ID)
		require.NoError(t, err)
		assert.Equal(t, todo.Value.Name, readResult.Name)
		assert.Equal(t, todo.Value.Status, readResult.Status)

		updateResult, err := model.Update(ctx, sampleTodo.ID, "complete")
		require.NoError(t, err)
		assert.Equal(t, Complete, updateResult.Status)
		assert.True(t, updateResult.CreatedDate.Before(updateResult.UpdatedDate))

		require.NoError(t, model.Del(ctx, sampleTodo.ID))
	})

	t.Run("Create and read multiple todos", func(t *testing.T) {
		names := []string{
			"Take out the trash",
			"Vacuum downstairs",
			"Fold the laundry",
		}

		for _, name := range names {
			_, err := model.Create(ctx, "", name)
			require.NoError(t, err)
		}

		allTodos, err := model.All(ctx)
		require.NoError(t, err)
		assert.Len(t, allTodos.Documents, len(names))
		assert.Equal(t, int64(len(names)), allTodos.Total)
	})

	t.Run("Search with no filters returns all todos", func(t *testing.T) {
		require.NoError(t, model.DelAll(ctx))

		first, err := model.Create(ctx, "", "Buy groceries")
		require.NoError(t, err)
		_, err = model.Update(ctx, first.ID, "in progress")
		require.NoError(t, err)
		_, err = model.Create(ctx, "", "Walk dog")
		require.NoError(t, err)

		results, err := model.Search(ctx, "", "")
		require.NoError(t, err)
		assert.Equal(t, int64(2), results.Total)
	})

	t.Run("Search by status handles spaces", func(t *testing.T) {
		require.NoError(t, model.DelAll(ctx))

		first, err := model.Create(ctx, "", "Buy groceries")
		require.NoError(t, err)
		_, err = model.Update(ctx, first.ID, "in progress")
		require.NoError(t, err)
		_, err = model.Create(ctx, "", "Walk dog")
		require.NoError(t, err)

		results, err := model.Search(ctx, "", "in progress")
		require.NoError(t, err)
		assert.Equal(t, int64(1), results.Total)
		assert.Equal(t, "Buy groceries", results.Documents[0].Value.Name)
	})
}
