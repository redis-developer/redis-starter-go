package todos

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Router struct {
	service Service
}

func (c Router) All(ctx echo.Context) error {
  rCtx := ctx.Request().Context()

  todos, err := c.service.All(rCtx)

	if err != nil {
		return ctx.String(http.StatusNotFound, err.Error())
	}

	return ctx.JSON(http.StatusOK, todos)
}

func (c Router) Search(ctx echo.Context) error {
  rCtx := ctx.Request().Context()
  name := ctx.QueryParam("name")
  status := ctx.QueryParam("status")
  todos, err := c.service.Search(rCtx, name, status)

  if err != nil {
    return ctx.String(http.StatusNotFound, err.Error())
  }

  return ctx.JSON(http.StatusOK, todos)
}

func (c Router) One(ctx echo.Context) error {
	rCtx := ctx.Request().Context()
	id := ctx.Param("id")
	todo, err := c.service.One(rCtx, id)

	if err != nil {
		return ctx.String(http.StatusNotFound, err.Error())
	}

	return ctx.JSON(http.StatusOK, todo)
}

type CreateTodoDTO struct {
  ID string `json:"id" form:"id"`
  Name  string `json:"name" form:"name"`
}

func (c Router) Create(ctx echo.Context) error {
  t := new(CreateTodoDTO)

  if err := ctx.Bind(t); err != nil {
    return ctx.String(http.StatusBadRequest, "bad request")
  }

	rCtx := ctx.Request().Context()
  todo, err := c.service.Create(rCtx, t.ID, t.Name)

  if err != nil {
    return ctx.String(http.StatusBadRequest, "bad request, todo not created")
  }

  return ctx.JSON(http.StatusOK, todo)
}

type UpdateTodoDTO struct {
  Status string `json:"status" form:"status"`
}

func (c Router) Update(ctx echo.Context) error {
  t := new(UpdateTodoDTO)

  if err := ctx.Bind(t); err != nil {
    return ctx.String(http.StatusBadRequest, "bad request")
  }

	rCtx := ctx.Request().Context()
  id := ctx.Param("id")
  todo, err := c.service.Update(rCtx, id, t.Status)

  if err != nil {
    return ctx.String(http.StatusNotFound, err.Error())
  }

  return ctx.JSON(http.StatusOK, todo)
}

func (c Router) Del(ctx echo.Context) error {
  rCtx := ctx.Request().Context()
  id := ctx.Param("id")

  err := c.service.Del(rCtx, id)

  if err != nil {
    return ctx.String(http.StatusBadRequest, err.Error())
  }

  return ctx.String(http.StatusOK, "OK")
}

func NewRouter(g *echo.Group, service Service) *Router {
	router := &Router{service: service}

  g.GET("", router.All)
	g.GET("/:id", router.One)
  g.GET("/search", router.Search)
  g.POST("", router.Create)
  g.PATCH("/:id", router.Update)
  g.DELETE("/:id", router.Del)

	g.RouteNotFound("/*", func(c echo.Context) error { return c.NoContent(http.StatusNotFound) })

	return router
}
