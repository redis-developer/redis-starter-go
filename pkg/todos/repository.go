package todos

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type TodosRepository struct {
	db *redis.Client
}

func haveIndex(ctx context.Context, db *redis.Client) bool {
	indexes := db.FT_List(ctx)

	indexes.Val()

	return slices.Contains(indexes.Val(), TodosIndex)
}

func parseTodoStr(id string, todoStr string) Todo {
	todo := Todo{ID: id}

	json.Unmarshal([]byte(todoStr), &todo)

	return todo
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
			As:        "name",
			FieldType: redis.SearchFieldTypeText,
		},
		&redis.FieldSchema{
			FieldName: "$.status",
			As:        "status",
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

func (c TodosRepository) All(ctx context.Context) (*Todos, error) {
	todosResult, err := c.db.FTSearch(ctx, TodosIndex, "*").Result()

	var documents = []Todo{}

	for _, todoDoc := range todosResult.Docs {
		documents = append(documents, parseTodoStr(todoDoc.ID, todoDoc.Fields["$"]))
	}

	return &Todos{
		Total:     int64(todosResult.Total),
		Documents: documents,
	}, err
}

func (c TodosRepository) One(ctx context.Context, id string) (*Todo, error) {
	todoStr, err := c.db.JSONGet(ctx, id).Result()

	if err != nil {
		return nil, err
	}

	todo := parseTodoStr(id, todoStr)

	return &todo, err
}

func (c TodosRepository) Search(ctx context.Context, name string, status string) (*Todos, error) {
	var searches []string

	log.Println(name)
	log.Println(status)

	if len(name) > 0 {
		searches = append(searches, fmt.Sprintf("@name:%s", name))
	}

	if len(status) > 0 {
		searches = append(searches, fmt.Sprintf("@status:%s", status))
	}

	log.Println(searches)

	todosResult, err := c.db.FTSearch(ctx, TodosIndex, strings.Join(searches, " ")).Result()

	var documents = []Todo{}

	for _, todoDoc := range todosResult.Docs {
		documents = append(documents, parseTodoStr(todoDoc.ID, todoDoc.Fields["$"]))
	}

	return &Todos{
		Total:     int64(todosResult.Total),
		Documents: documents,
	}, err
}

func (c TodosRepository) Create(ctx context.Context, id string, name string) (*Todo, error) {
	now := time.Now()

  todo := &Todo{
    ID: id,
    Name: name,
    Status: NotStarted,
    CreatedDate: now,
    UpdatedDate: now,
  }

  _, err := c.db.JSONSet(ctx, id, "$", todo).Result();

  if err != nil {
    return nil, err
  }

  return todo, nil
}

func (c TodosRepository) Update(ctx context.Context, id string, status TodoStatus) (*Todo, error) {
  todo, err := c.One(ctx, id)

  if err != nil {
    return nil, err
  }

  todo.Status = status
  todo.UpdatedDate = time.Now()

  _, err = c.db.JSONSet(ctx, id, "$", todo).Result();

  if err != nil {
    return nil, err
  }

  return todo, nil
}

func (c TodosRepository) Del(ctx context.Context, id string) error {
  _, err := c.db.JSONDel(ctx, id, "$").Result()

  return err
}

func (c TodosRepository) DelAll(ctx context.Context) error {
  allTodos, err := c.All(ctx)

  if err != nil {
    return err
  }

  for _, todo := range(allTodos.Documents) {
    err = c.Del(ctx, todo.ID)

    if err != nil {
      return err
    }
  }

  return nil
}

func NewRepository(db *redis.Client) *TodosRepository {
	repository := &TodosRepository{
		db: db,
	}

	repository.CreateIndexIfNotExists(context.Background())

	return repository
}
