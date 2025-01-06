package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis-developer/redis-starter-go/pkg/redis"
	"github.com/redis-developer/redis-starter-go/pkg/todos"
)

func SetupApp(config Config) *echo.Echo {
	database := redis.GetClient(config.RedisUrl())

	e := echo.New()
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339} ${status} ${method} ${uri} ${latency_human} " +
			"[${user_agent} ${remote_ip}]\n",
	}))
	e.Use(middleware.Recover())

	apiGroup := e.Group("/api")
	apiGroup.RouteNotFound("/*", func(c echo.Context) error {
		return c.NoContent(http.StatusNotFound)
	})

	todos.NewRouter(apiGroup.Group("/todos"), todos.NewStore(database))

	return e
}

