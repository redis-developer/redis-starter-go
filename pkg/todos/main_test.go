package todos

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis-developer/redis-starter-go/cmd/config"
	"github.com/redis-developer/redis-starter-go/pkg/redis"
)

func printTodo(todo *Todo) string {
	return fmt.Sprintf("id:%s name:%s status:%s", todo.ID, todo.Name, todo.Status)
}

func todosEqual(t1 *Todo, t2 *Todo) bool {
	return t1.ID == t2.ID && t1.Name == t2.Name && t1.Status == t2.Status
}

func TestCrud(t *testing.T) {
	cfg := config.Config{}
	database := redis.GetClient(cfg.REDIS_URL())
	repository := NewRepository(database)
	service := NewService(repository)
	ctx := context.Background()
	repository.CreateIndexIfNotExists(ctx)

	t.Cleanup(func() {
		repository.DelAll(ctx)
		repository.DropIndex(ctx)
		repository.CreateIndexIfNotExists(ctx)
	})

	t.Run("CRUD for a single todo", func(t *testing.T) {
		sampleTodo := &Todo{
			Name:   "Take out the trash",
			ID:     "todos:abc123",
			Status: "todo",
		}
		todo, err := service.Create(ctx, sampleTodo.ID, sampleTodo.Name)

		if err != nil {
			t.Errorf("todo not created: %s", err.Error())
			return
		}

		if !todosEqual(todo, sampleTodo) {
			t.Errorf("got %s, want %s", printTodo(todo), printTodo(sampleTodo))
			return
		}

		readResult, err := service.One(ctx, todo.ID)

		if err != nil {
			t.Errorf("todo not read: %s", err.Error())
			return
		}

		if !todosEqual(readResult, todo) {
			t.Errorf("got %s, want %s", printTodo(readResult), printTodo(todo))
			return
		}

		updateResult, err := service.Update(ctx, sampleTodo.ID, "complete")

		if err != nil {
			t.Errorf("todo not updated: %s", err.Error())
			return
		}

		if updateResult.Status != "complete" {
			t.Errorf("got status:%s, want status:%s", updateResult.Status, "complete")
			return
		}

		if updateResult.CreatedDate.After(updateResult.UpdatedDate) {
			t.Errorf("got updated_date:%s, want after:%s",
        updateResult.UpdatedDate.Format(time.RFC3339),
        updateResult.CreatedDate.Format(time.RFC3339))
      return
    }

    service.Del(ctx, updateResult.ID)
	})
}
