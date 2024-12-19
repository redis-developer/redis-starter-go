package todos

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const TodosIndex = "todos-idx"
const TodosPrefix = "todos:"

type TodoStatus string

const (
	NotStarted TodoStatus = "todo"
	InProgress TodoStatus = "in progress"
	Complete   TodoStatus = "complete"
)

var TodoStatusMap = map[string]TodoStatus{
	"todo":        NotStarted,
	"in progress": InProgress,
	"complete":    Complete,
}

type Todo struct {
	ID     string     `json:"id"`
	Name   string     `json:"name"`
	Status TodoStatus `json:"status"`

	CreatedDate time.Time `json:"created_date"`
	UpdatedDate time.Time `json:"updated_date"`
}

type Todos struct {
	Total     int64  `json:"total"`
	Documents []Todo `json:"documents"`
}

type TodoStore interface {
	CreateIndexIfNotExists(ctx context.Context) error
	DropIndex(ctx context.Context)
	All(ctx context.Context) (*Todos, error)
	One(ctx context.Context, id string) (*Todo, error)
	Search(ctx context.Context, name string, status string) (*Todos, error)
	Create(ctx context.Context, id string, name string) (*Todo, error)
	Update(ctx context.Context, id string, status string) (*Todo, error)
	Del(ctx context.Context, id string) error
	DelAll(ctx context.Context) error
}

type store struct {
	db *redis.Client
}

func (c store) haveIndex(ctx context.Context) bool {
	indexes := c.db.FT_List(ctx)

	indexes.Val()

	return slices.Contains(indexes.Val(), TodosIndex)
}

func parseTodoStr(id string, todoStr string) Todo {
	todo := Todo{ID: id}

	json.Unmarshal([]byte(todoStr), &todo)

	return todo
}

func formatId(id string) (string, error) {
	matched, err := regexp.Match(`^todos:`, []byte(id))

	if err != nil {
		return id, fmt.Errorf("failed to format id: %w", err)
	}

	if matched {
		return id, err
	}

	return fmt.Sprintf("todos:%s", id), err
}

func (c store) CreateIndexIfNotExists(ctx context.Context) error {
	if c.haveIndex(ctx) {
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

	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return err
}

func (c store) DropIndex(ctx context.Context) {
	if !c.haveIndex(ctx) {
		return
	}

	c.db.FTDropIndex(ctx, TodosIndex)
}

func (c store) All(ctx context.Context) (*Todos, error) {
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

func (c store) One(ctx context.Context, id string) (*Todo, error) {
	fId, err := formatId(id)

	if err != nil {
		return nil, fmt.Errorf("failed to get normalized id: %w", err)
	}

	todoStr, err := c.db.JSONGet(ctx, fId).Result()

	if err != nil {
		return nil, fmt.Errorf("failed JSON.GET for todo: %w", err)
	}

	todo := parseTodoStr(fId, todoStr)

	return &todo, err
}

func (c store) Search(
	ctx context.Context,
	name string,
	status string) (*Todos, error) {
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

	todosResult, err := c.db.FTSearch(
		ctx,
		TodosIndex,
		strings.Join(searches, " "),
	).Result()

	if err != nil {
		return nil, fmt.Errorf("failed FT.SEARCH for todos: %w", err)
	}

	var documents = []Todo{}

	for _, todoDoc := range todosResult.Docs {
		documents = append(documents, parseTodoStr(todoDoc.ID, todoDoc.Fields["$"]))
	}

	return &Todos{
		Total:     int64(todosResult.Total),
		Documents: documents,
	}, err
}

func (c store) Create(
	ctx context.Context,
	id string,
	name string) (*Todo, error) {
	now := time.Now()
	fId, err := formatId(id)

	if err != nil {
		return nil, fmt.Errorf("failed to get normalized id: %w", err)
	}

	todo := &Todo{
		ID:          fId,
		Name:        name,
		Status:      NotStarted,
		CreatedDate: now,
		UpdatedDate: now,
	}

	_, err = c.db.JSONSet(ctx, fId, "$", todo).Result()

	if err != nil {
		return nil, fmt.Errorf("failed JSON.SET for todo: %w", err)
	}

	return todo, nil
}

func (c store) Update(
	ctx context.Context,
	id string,
	status string) (*Todo, error) {
	fId, err := formatId(id)

	if err != nil {
		return nil, fmt.Errorf("failed to get normalized id: %w", err)
	}

	todoStatus, ok := TodoStatusMap[status]

	if !ok {
		return nil, fmt.Errorf("Invalid status %s", todoStatus)
	}

	todo, err := c.One(ctx, fId)

	if err != nil {
		return nil, fmt.Errorf("failed to update todo, not found: %w", err)
	}

	todo.Status = todoStatus
	todo.UpdatedDate = time.Now()

	_, err = c.db.JSONSet(ctx, fId, "$", todo).Result()

	if err != nil {
		return nil, fmt.Errorf("failed JSON.SET for todo: %w", err)
	}

	return todo, nil
}

func (c store) Del(ctx context.Context, id string) error {
	fId, err := formatId(id)

	if err != nil {
		return fmt.Errorf("failed to get normalized id: %w", err)
	}

	_, err = c.db.JSONDel(ctx, fId, "$").Result()

	return err
}

func (c store) DelAll(ctx context.Context) error {
	allTodos, err := c.All(ctx)

	if err != nil {
		return fmt.Errorf("failed to find all todos: %w", err)
	}

	for _, todo := range allTodos.Documents {
		err = c.Del(ctx, todo.ID)

		if err != nil {
			return fmt.Errorf("failed to delete todo: %w", err)
		}
	}

	return nil
}

func NewStore(db *redis.Client) *store {
	repository := &store{
		db: db,
	}

	repository.CreateIndexIfNotExists(context.Background())

	return repository
}
