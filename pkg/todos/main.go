package todos

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"
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

var TodoStatusMap = map[string]TodoStatus {
  "todo": NotStarted,
  "in progress": InProgress,
  "complete": Complete,
}

type Todo struct {
	ID     string     `json:"id"`
	Name   string     `json:"name"`
	Status TodoStatus `json:"status"`

	CreatedDate time.Time `json:"created_date"`
	UpdatedDate time.Time `json:"updated_date"`
}

type Todos struct {
	Total int64  `json:"total"`
	Documents  []Todo `json:"documents"`
}

type Repository interface {
	CreateIndexIfNotExists(ctx context.Context) error
	DropIndex(ctx context.Context)
	All(ctx context.Context) (*Todos, error)
	One(ctx context.Context, id string) (*Todo, error)
	Search(ctx context.Context, name string, status string) (*Todos, error)
	Create(ctx context.Context, id string, name string) (*Todo, error)
	Update(ctx context.Context, id string, status TodoStatus) (*Todo, error)
  Del(ctx context.Context, id string) error
  DelAll(ctx context.Context) error
}

type Service interface {
	All(ctx context.Context) (*Todos, error)
	One(ctx context.Context, id string) (*Todo, error)
	Search(ctx context.Context, name string, status string) (*Todos, error)
	Create(ctx context.Context, id string, name string) (*Todo, error)
	Update(ctx context.Context, id string, status string) (*Todo, error)
  Del(ctx context.Context, id string) error
  DelAll(ctx context.Context) error
}

type TodosComponent struct {
	repository Repository
	service    Service
	router     Router
}

func NewComponent(g *echo.Group, redis *redis.Client) *TodosComponent {
	repository := NewRepository(redis)
	service := NewService(repository)
	router := NewRouter(g, service)

	return &TodosComponent{
		repository: *repository,
		service:    *service,
		router:     *router,
	}
}
