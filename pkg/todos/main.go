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

type Todo struct {
	Name   string     `json:"name"`
	Status TodoStatus `json:"status"`

	CreatedDate time.Time `json:"created_date"`
	UpdatedDate time.Time `json:"updated_date"`
}

type Repository interface {
	One(ctx context.Context, id string) (*Todo, error)
}

type Service interface {
	One(ctx context.Context, id string) (*Todo, error)
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
