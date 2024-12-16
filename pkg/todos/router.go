package todos

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Router struct {
	service Service
}

func (r *Router) One(ctx echo.Context) error {
	rCtx := ctx.Request().Context()
	id := ctx.Param("id")
	exercise, err := r.service.One(rCtx, id)

	if err != nil {
		return ctx.String(http.StatusNotFound, err.Error())
	}

	return ctx.JSON(http.StatusOK, exercise)
}

func NewRouter(g *echo.Group, service Service) *Router {
	r := &Router{service: service}

	g.GET("/:id", r.One)

	g.RouteNotFound("/*", func(c echo.Context) error { return c.NoContent(http.StatusNotFound) })

	return r
}
