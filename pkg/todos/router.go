/*
Package todos provides a component API for all the CRUD operations around building a
todo list.
*/
package todos

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Router provides the methods necessary for facilitating CRUD using an echo API
type Router interface {
	All(ctx echo.Context) error
	Search(ctx echo.Context) error
	One(ctx echo.Context) error
	Create(ctx echo.Context) error
	Update(ctx echo.Context) error
	Del(ctx echo.Context) error
}

// Implements a Router with a provided Store
type TodoRouter struct {
	store Store
}

// All handles the route for getting all Todos
func (c TodoRouter) All(ctx echo.Context) error {
	rCtx := ctx.Request().Context()

	todos, err := c.store.All(rCtx)

	if err != nil {
		slog.Debug(err.Error())
		return ctx.String(http.StatusNotFound, "not found")
	}

	return ctx.JSON(http.StatusOK, todos)
}

// Search handles the route for searching all Todos
func (c TodoRouter) Search(ctx echo.Context) error {
	rCtx := ctx.Request().Context()
	name := ctx.QueryParam("name")
	status := ctx.QueryParam("status")
	todos, err := c.store.Search(rCtx, name, status)

	if err != nil {
		slog.Debug(err.Error())
		return ctx.String(http.StatusNotFound, "not found")
	}

	return ctx.JSON(http.StatusOK, todos)
}

// One handles the route for getting a single Todo
func (c TodoRouter) One(ctx echo.Context) error {
	rCtx := ctx.Request().Context()
	id := ctx.Param("id")
	todo, err := c.store.One(rCtx, id)

	if err != nil {
		slog.Debug(err.Error())
		return ctx.String(http.StatusNotFound, "not found")
	}

	return ctx.JSON(http.StatusOK, todo)
}

// CreateTodoDTO handles parsing JSON requests for creating Todos
type CreateTodoDTO struct {
	ID   string `json:"id" form:"id"`
	Name string `json:"name" form:"name"`
}

// Create handles the route for creating a Todo
func (c TodoRouter) Create(ctx echo.Context) error {
	t := new(CreateTodoDTO)

	if err := ctx.Bind(t); err != nil {
		slog.Debug(err.Error())
		return ctx.String(http.StatusBadRequest, "bad request")
	}

	rCtx := ctx.Request().Context()
	todo, err := c.store.Create(rCtx, t.ID, t.Name)

	if err != nil {
		slog.Debug(err.Error())
		return ctx.String(http.StatusBadRequest, "bad request, todo not created")
	}

	return ctx.JSON(http.StatusOK, todo)
}

// UpdateTodoDTO handles parsing JSON requests for updating Todos
type UpdateTodoDTO struct {
	Status string `json:"status" form:"status"`
}

// Update handles the route for updating a Todo
func (c TodoRouter) Update(ctx echo.Context) error {
	t := new(UpdateTodoDTO)

	if err := ctx.Bind(t); err != nil {
		slog.Debug(err.Error())
		return ctx.String(http.StatusBadRequest, "bad request")
	}

	rCtx := ctx.Request().Context()
	id := ctx.Param("id")
	todo, err := c.store.Update(rCtx, id, t.Status)

	if err != nil {
		slog.Debug(err.Error())
		return ctx.String(http.StatusNotFound, "not found")
	}

	return ctx.JSON(http.StatusOK, todo)
}

// Del handles the route for deleting a Todo
func (c TodoRouter) Del(ctx echo.Context) error {
	rCtx := ctx.Request().Context()
	id := ctx.Param("id")

	err := c.store.Del(rCtx, id)

	if err != nil {
		slog.Debug(err.Error())
		return ctx.String(http.StatusBadRequest, "not found")
	}

	return ctx.String(http.StatusOK, "ok")
}

// NewRouter returns a Router that uses the passed in echo group and Store
// to create an API for Todo CRUD
func NewRouter(g *echo.Group, store Store) Router {
	router := &TodoRouter{store: store}

	g.GET("", router.All)
	g.GET("/:id", router.One)
	g.GET("/search", router.Search)
	g.POST("", router.Create)
	g.PATCH("/:id", router.Update)
	g.DELETE("/:id", router.Del)

	g.RouteNotFound("/*", func(c echo.Context) error {
		return c.NoContent(http.StatusNotFound)
	})

	return router
}
