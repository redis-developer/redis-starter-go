package todos

import (
	"context"
	"testing"

	"github.com/redis-developer/redis-starter-go/cmd/config"
	"github.com/redis-developer/redis-starter-go/pkg/redis"
	"github.com/stretchr/testify/assert"
)

func todosEqual(t *testing.T, expected *Todo, actual *Todo) {
	assert.Equal(t, expected.ID, actual.ID)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Status, actual.Status)
}

func TestCrud(t *testing.T) {
	cfg := config.Config{}
	database := redis.GetClient(cfg.REDIS_URL())
	store := NewStore(database)
	ctx := context.Background()
	store.CreateIndexIfNotExists(ctx)
	store.DelAll(ctx)

	t.Cleanup(func() {
		store.DelAll(ctx)
		store.DropIndex(ctx)
		store.CreateIndexIfNotExists(ctx)
	})

	t.Run("CRUD for a single todo", func(t *testing.T) {
		sampleTodo := &Todo{
			Name:   "Take out the trash",
			ID:     "todos:abc123",
			Status: NotStarted,
		}
		todo, err := store.Create(ctx, sampleTodo.ID, sampleTodo.Name)

		sampleTodo.ID = todo.ID

		if assert.NoErrorf(t, err, "todo not created: %s", "formatted") {
			todosEqual(t, sampleTodo, todo)
		}

		readResult, err := store.One(ctx, todo.ID)

		if assert.NoErrorf(t, err, "todo not read: %s", "formatted") {
			todosEqual(t, todo, readResult)
		}

		updateResult, err := store.Update(ctx, sampleTodo.ID, "complete")

		if assert.NoErrorf(t, err, "todo not updated: %s", "formatted") {
			assert.Equal(t, Complete, updateResult.Status)
			assert.True(t, updateResult.CreatedDate.Before(updateResult.UpdatedDate))
		}

		err = store.Del(ctx, updateResult.ID)

		assert.NoErrorf(t, err, "todo not deleted: %s", "formatted")
	})

	t.Run("Create and read multiple todos", func(t *testing.T) {
		todos := []string{
			"Take out the trash",
			"Vacuum downstairs",
			"Fold the laundry",
		}

		for _, todo := range todos {
			_, err := store.Create(ctx, "", todo)

			assert.NoErrorf(t, err, "error creating todo: %s", "formatted")
		}

		allTodos, err := store.All(ctx)

		if assert.NoErrorf(t, err, "error getting all todos: %s", "formatted") {
			assert.Equal(t, len(todos), len(allTodos.Documents))
			assert.True(t, len(allTodos.Documents) == int(allTodos.Total))
		}

		for _, todo := range allTodos.Documents {
			assert.Contains(t, todos, todo.Name)
		}
	})
}
