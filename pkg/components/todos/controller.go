package todos

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Controller provides the application-level behavior for Todo CRUD.
type Controller interface {
	All(ctx context.Context) (*Todos, error)
	Search(ctx context.Context, request SearchTodosRequest) (*Todos, error)
	One(ctx context.Context, id string) (*Todo, error)
	Create(ctx context.Context, request CreateTodoRequest) (*TodoDocument, error)
	Update(ctx context.Context, id string, request UpdateTodoRequest) (*Todo, error)
	Del(ctx context.Context, id string) error
}

// TodoController implements a Controller with a provided Model.
type TodoController struct {
	model Model
	log   *slog.Logger
}

func toHTTPError(err error) error {
	var todoError *TodoError
	if errors.As(err, &todoError) {
		switch todoError.ErrorType {
		case NotFound:
			return echo.NewHTTPError(http.StatusNotFound, todoError.ClientMessage)
		case Invalid:
			return echo.NewHTTPError(http.StatusBadRequest, todoError.ClientMessage)
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, todoError.ClientMessage)
		}
	}

	return err
}

func (c TodoController) All(ctx context.Context) (*Todos, error) {
	c.log.Debug("fetching all todos")
	todos, err := c.model.All(ctx)
	if err != nil {
		return nil, toHTTPError(err)
	}

	return todos, nil
}

func (c TodoController) Search(ctx context.Context, request SearchTodosRequest) (*Todos, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	c.log.Debug("searching todos", slog.String("name", request.Name), slog.String("status", request.Status))
	todos, err := c.model.Search(ctx, request.Name, request.Status)
	if err != nil {
		return nil, toHTTPError(err)
	}

	return todos, nil
}

func (c TodoController) One(ctx context.Context, id string) (*Todo, error) {
	validatedID, err := ValidateTodoID(id)
	if err != nil {
		return nil, err
	}

	c.log.Debug("fetching todo", slog.String("id", validatedID))
	todo, err := c.model.One(ctx, validatedID)
	if err != nil {
		return nil, toHTTPError(err)
	}

	return todo, nil
}

func (c TodoController) Create(ctx context.Context, request CreateTodoRequest) (*TodoDocument, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	todo, err := c.model.Create(ctx, request.ID, request.Name)
	if err != nil {
		return nil, toHTTPError(err)
	}

	c.log.Debug("creating todo", slog.String("id", todo.ID), slog.String("name", request.Name))
	return todo, nil
}

func (c TodoController) Update(ctx context.Context, id string, request UpdateTodoRequest) (*Todo, error) {
	validatedID, err := ValidateTodoID(id)
	if err != nil {
		return nil, err
	}

	if err := request.Validate(); err != nil {
		return nil, err
	}

	todo, err := c.model.Update(ctx, validatedID, request.Status)
	if err != nil {
		return nil, toHTTPError(err)
	}

	c.log.Debug("updating todo", slog.String("id", validatedID), slog.String("status", request.Status))
	return todo, nil
}

func (c TodoController) Del(ctx context.Context, id string) error {
	validatedID, err := ValidateTodoID(id)
	if err != nil {
		return err
	}

	if err := c.model.Del(ctx, validatedID); err != nil {
		return toHTTPError(err)
	}

	c.log.Debug("deleting todo", slog.String("id", validatedID))
	return nil
}

func NewController(model Model, logger *slog.Logger) *TodoController {
	return &TodoController{
		model: model,
		log:   logger.With("component", "todos"),
	}
}
