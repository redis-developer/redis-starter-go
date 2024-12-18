package todos

import (
	"context"
	"fmt"
	"testing"

	"github.com/redis-developer/redis-starter-go/cmd/config"
	"github.com/redis-developer/redis-starter-go/pkg/redis"
)

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
		name := "Take out the trash"
		id := "abc123"
		todo, err := service.Create(ctx, id, name)

		if err != nil {
			t.Errorf("todo not created: %s", err.Error())
			return
		}

		if todo.Name != name || todo.ID != fmt.Sprintf("todos:%s", id) {
			t.Errorf("got id:%s name:%s, want id:%s name:%s", todo.ID, todo.Name, id, name)
		}
	})
}
