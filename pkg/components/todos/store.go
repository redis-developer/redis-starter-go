package todos

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// The redis index name for Todos
const TodoIndex = "todos-idx"

// The key prefix for Todos stored in redis
const TodoPrefix = "todos:"

// Enumerator for all the status types a Todo can have
type TodoStatus string

// A Todo has one of these status values
const (
	NotStarted TodoStatus = "todo"
	InProgress TodoStatus = "in progress"
	Complete   TodoStatus = "complete"
)

// Maps a string to its corresponding TodoStatus
var TodoStatusMap = map[string]TodoStatus{
	"todo":        NotStarted,
	"in progress": InProgress,
	"complete":    Complete,
}

// A Todo is the base model used to store the name and status of Todos
type Todo struct {
	Name   string     `json:"name"`
	Status TodoStatus `json:"status"`

	CreatedDate time.Time `json:"created_date"`
	UpdatedDate time.Time `json:"updated_date"`
}

// A TodoDocument contains the Todo's ID in Redis as well as the Todo value
type TodoDocument struct {
	ID    string `json:"id"`
	Value Todo   `json:"value"`
}

// A Todos object is the result of list-based operations against redis
type Todos struct {
	Total     int64          `json:"total"`
	Documents []TodoDocument `json:"documents"`
}

// Store provides the methods necessary to handle CRUD operations for Todos.
// It also handles creating the index for searching Todos.
type Store interface {
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

// Implements a Store with a provided redis client
type TodoStore struct {
	db *redis.Client
}

// Enumerator for the types of internal errors thrown
type TodoErrorType string

// A TodoError has one of these types
const (
	NotFound TodoErrorType = "not found"
	Invalid  TodoErrorType = "invalid"
	Unknown  TodoErrorType = "unknown"
)

// A TodoError is a Redis-level error with a safe message for the client
type TodoError struct {
	ErrorType     TodoErrorType
	ClientMessage string
	Err           error
}

// Error is implemented to make a TodoError operate as a go error
func (c *TodoError) Error() string {
	if c.Err == nil {
		return c.ClientMessage
	}

	return c.Err.Error()
}

// parseTodoStr returns a Todo object based on the input JSON string
func parseTodoStr(todoJson string) Todo {
	todo := Todo{}

	json.Unmarshal([]byte(todoJson), &todo)

	return todo
}

// formatId returns a normalized ID string that allows IDs to either
// include or exclude the TodosPrefix
func formatId(id string) string {
	matched, err := regexp.Match(`^`+TodoPrefix, []byte(id))

	if err != nil {
		// This should not happen, if it does that is bad
		panic(err)
	}

	if matched {
		return id
	}

	return fmt.Sprintf("%s%s", TodoPrefix, id)
}

// haveIndex returns whether or not the Todo index already exists in redis
func (c TodoStore) haveIndex(ctx context.Context) bool {
	indexes := c.db.FT_List(ctx)

	return slices.Contains(indexes.Val(), TodoIndex)
}

// CreateIndexIfNotExists ensures that the Todos index exists in redis
func (c TodoStore) CreateIndexIfNotExists(ctx context.Context) error {
	if c.haveIndex(ctx) {
		return nil
	}

	_, err := c.db.FTCreate(
		ctx,
		TodoIndex,
		&redis.FTCreateOptions{
			OnJSON: true,
			Prefix: []interface{}{TodoPrefix},
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
		return &TodoError{
			ErrorType:     Unknown,
			ClientMessage: "failed to create index",
			Err:           fmt.Errorf("failed to create index: %w", err),
		}
	}

	return err
}

// DropIndex will delete the Todos index in redis
func (c TodoStore) DropIndex(ctx context.Context) {
	if !c.haveIndex(ctx) {
		return
	}

	c.db.FTDropIndex(ctx, TodoIndex)
}

// All returns a Todos object that contains all existing Todos in redis
func (c TodoStore) All(ctx context.Context) (*Todos, error) {
	todosResult, err := c.db.FTSearch(ctx, TodoIndex, "*").Result()

	if err != nil {
		return nil, &TodoError{
			ErrorType:     Unknown,
			ClientMessage: "failed to get all todos",
			Err:           fmt.Errorf("failed to get all todos: %w", err),
		}
	}

	var documents = []TodoDocument{}

	for _, todoDoc := range todosResult.Docs {
		todo := parseTodoStr(todoDoc.Fields["$"])
		documents = append(documents, TodoDocument{
			ID:    todoDoc.ID,
			Value: todo,
		})
	}

	return &Todos{
		Total:     int64(todosResult.Total),
		Documents: documents,
	}, err
}

// One returns a Todo if it exists in redis based on the input id
func (c TodoStore) One(ctx context.Context, id string) (*Todo, error) {
	fId := formatId(id)

	todoStr, err := c.db.JSONGet(ctx, fId).Result()

	if err != nil {
		return nil, &TodoError{
			ErrorType:     Unknown,
			ClientMessage: "failed to get todo",
			Err:           fmt.Errorf("failed to get todo: %w", err),
		}
	}

	if len(todoStr) == 0 {
		return nil, &TodoError{
			ErrorType:     NotFound,
			ClientMessage: "todo not found",
			Err:           err,
		}
	}

	todo := parseTodoStr(todoStr)

	return &todo, err
}

// Search returns a Todos object with a list of Todos that match the
// input paramters (name and/or status)
func (c TodoStore) Search(
	ctx context.Context,
	name string,
	status string) (*Todos, error) {
	var searches []string

	if len(name) > 0 {
		searches = append(searches, fmt.Sprintf("@name:%s", name))
	}

	if len(status) > 0 {
		searches = append(searches, fmt.Sprintf("@status:%s", status))
	}

	todosResult, err := c.db.FTSearch(
		ctx,
		TodoIndex,
		strings.Join(searches, " "),
	).Result()

	if err != nil {
		return nil, &TodoError{
			ErrorType:     Unknown,
			ClientMessage: "failed search for todos",
			Err:           fmt.Errorf("failed to search for todos: %w", err),
		}
	}

	var documents = []TodoDocument{}

	for _, todoDoc := range todosResult.Docs {
		todo := parseTodoStr(todoDoc.Fields["$"])
		documents = append(documents, TodoDocument{
			ID:    todoDoc.ID,
			Value: todo,
		})
	}

	return &Todos{
		Total:     int64(todosResult.Total),
		Documents: documents,
	}, err
}

// Create returns a newly created Todo from redis based on the input ID and name
func (c TodoStore) Create(
	ctx context.Context,
	id string,
	name string) (*TodoDocument, error) {
	now := time.Now()

	if len(id) == 0 {
		id = uuid.New().String()
	}

	if len(name) == 0 {
		return nil, &TodoError{
			ErrorType:     Invalid,
			ClientMessage: "todo must have a name",
			Err:           nil,
		}
	}

	fId := formatId(id)

	todo := &TodoDocument{
		ID: fId,
		Value: Todo{
			Name:        name,
			Status:      NotStarted,
			CreatedDate: now,
			UpdatedDate: now,
		},
	}

	_, err := c.db.JSONSet(ctx, fId, "$", todo.Value).Result()

	if err != nil {
		return nil, &TodoError{
			ErrorType:     Invalid,
			ClientMessage: "failed to update todo",
			Err:           err,
		}
	}

	return todo, nil
}

// Update returns an updated Todo from redis based on the ID with a new status
func (c TodoStore) Update(
	ctx context.Context,
	id string,
	status string) (*Todo, error) {
	fId := formatId(id)

	todoStatus, ok := TodoStatusMap[status]

	if !ok {
		return nil, &TodoError{
			ErrorType:     Invalid,
			ClientMessage: fmt.Sprintf("invalid status %s", todoStatus),
			Err:           nil,
		}
	}

	todo, err := c.One(ctx, fId)

	if err != nil {
		return nil, &TodoError{
			ErrorType:     NotFound,
			ClientMessage: "todo not found",
			Err:           err,
		}
	}

	todo.Status = todoStatus
	todo.UpdatedDate = time.Now()

	_, err = c.db.JSONSet(ctx, fId, "$", todo).Result()

	if err != nil {
		return nil, &TodoError{
			ErrorType:     Unknown,
			ClientMessage: "failed to update todo",
			Err:           fmt.Errorf("failed to update todo: %w", err),
		}
	}

	return todo, nil
}

// Del deletes a todo in redis if it exists
func (c TodoStore) Del(ctx context.Context, id string) error {
	fId := formatId(id)

	_, err := c.db.JSONDel(ctx, fId, "$").Result()

	if err != nil {
		return &TodoError{
			ErrorType:     Unknown,
			ClientMessage: "failed to delete todo",
			Err:           fmt.Errorf("failed to delete todo: %w", err),
		}
	}

	return nil
}

// DelAll deletes all the Todos in redis
func (c TodoStore) DelAll(ctx context.Context) error {
	allTodos, err := c.All(ctx)

	if err != nil {
		return &TodoError{
			ErrorType:     Unknown,
			ClientMessage: "failed to find all todos",
			Err:           fmt.Errorf("failed to find all todos: %w", err),
		}
	}

	for _, todo := range allTodos.Documents {
		err = c.Del(ctx, todo.ID)

		if err != nil {
			return &TodoError{
				ErrorType:     Unknown,
				ClientMessage: "failed to delete todo",
				Err:           fmt.Errorf("failed to delete todo: %w", err),
			}
		}
	}

	return nil
}

// NewStore returns a Store that uses the passed-in redis client
// to manage Todos
func NewStore(db *redis.Client) *TodoStore {
	store := &TodoStore{
		db: db,
	}

	store.CreateIndexIfNotExists(context.Background())

	return store
}
