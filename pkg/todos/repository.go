package todos

import (
	"context"
	"encoding/json"
  "slices"

	"github.com/redis/go-redis/v9"
)

type TodosRepository struct {
	db *redis.Client
}

func haveIndex(ctx context.Context, db *redis.Client) bool {
  indexes := db.FT_List(ctx);

  indexes.Val()

  return slices.Contains(indexes.Val(), TodosIndex)
}

func (c TodosRepository) CreateIndexIfNotExists(ctx context.Context) error {
  if haveIndex(ctx, c.db) {
    return nil
  }

  _, err := c.db.FTCreate(
    ctx,
    TodosIndex,
    &redis.FTCreateOptions{
      OnJSON: true,
      Prefix: []interface{}{TodosPrefix},
    },
    &redis.FieldSchema{
      FieldName: "$.name",
      As: "name",
      FieldType: redis.SearchFieldTypeText,
    },
    &redis.FieldSchema{
      FieldName: "$.status",
      As: "status",
      FieldType: redis.SearchFieldTypeText,
    },
  ).Result()

  return err
}

func (c TodosRepository) DropIndex(ctx context.Context) {
  if !haveIndex(ctx, c.db) {
    return
  }

  c.db.FTDropIndex(ctx, TodosIndex)
}

func (c TodosRepository) One(ctx context.Context, id string) (*Todo, error) {
	todoStr, err := c.db.JSONGet(ctx, id).Result()

	if err != nil {
		return nil, err
	}

	todo := Todo{}

	json.Unmarshal([]byte(todoStr), &todo)

	return &todo, err
}

func NewRepository(db *redis.Client) *TodosRepository {
  repository := &TodosRepository{
		db: db,
	}

  repository.CreateIndexIfNotExists(context.Background())

  return repository
}
