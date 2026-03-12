/*
Package todos provides a component API for all the CRUD operations around building a
todo list.
*/
package todos

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Router provides the methods necessary for facilitating CRUD using an echo API.
type Router interface {
	All(ctx echo.Context) error
	Search(ctx echo.Context) error
	One(ctx echo.Context) error
	Create(ctx echo.Context) error
	Update(ctx echo.Context) error
	Del(ctx echo.Context) error
}

// TodoRouter implements a Router with a provided Controller.
type TodoRouter struct {
	controller Controller
}

// All handles the route for getting all Todos.
func (r TodoRouter) All(ctx echo.Context) error {
	todos, err := r.controller.All(ctx.Request().Context())
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, todos)
}

// One handles the route for getting a single Todo.
func (r TodoRouter) One(ctx echo.Context) error {
	todo, err := r.controller.One(ctx.Request().Context(), ctx.Param("id"))
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, todo)
}

// Search handles the route for searching all Todos.
func (r TodoRouter) Search(ctx echo.Context) error {
	todos, err := r.controller.Search(ctx.Request().Context(), SearchTodosRequest{
		Name:   ctx.QueryParam("name"),
		Status: ctx.QueryParam("status"),
	})
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, todos)
}

// Create handles the route for creating a Todo.
func (r TodoRouter) Create(ctx echo.Context) error {
	request := new(CreateTodoRequest)
	if err := ctx.Bind(request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	todo, err := r.controller.Create(ctx.Request().Context(), *request)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, todo)
}

// Update handles the route for updating a Todo.
func (r TodoRouter) Update(ctx echo.Context) error {
	request := new(UpdateTodoRequest)
	if err := ctx.Bind(request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	todo, err := r.controller.Update(ctx.Request().Context(), ctx.Param("id"), *request)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, todo)
}

// Del handles the route for deleting a Todo.
func (r TodoRouter) Del(ctx echo.Context) error {
	if err := r.controller.Del(ctx.Request().Context(), ctx.Param("id")); err != nil {
		return err
	}

	return ctx.String(http.StatusOK, "ok")
}

// NewRouter returns a Router that uses the passed in echo group and Controller
// to create an API for Todo CRUD.
func NewRouter(g *echo.Group, controller Controller) *TodoRouter {
	router := &TodoRouter{controller: controller}

	g.GET("", router.All)
	g.GET("/search", router.Search)
	g.GET("/:id", router.One)
	g.POST("", router.Create)
	g.PATCH("/:id", router.Update)
	g.DELETE("/:id", router.Del)

	g.RouteNotFound("/*", func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusNotFound)
	})

	return router
}
