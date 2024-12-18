package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis-developer/redis-starter-go/pkg/redis"
	"github.com/redis-developer/redis-starter-go/pkg/todos"
)

type Config interface {
	REDIS_URL() string
	PORT() string
}

type Server struct {
	port string
	e    *echo.Echo
}

func (s *Server) Run() {
	e := s.e

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	// Start server
	go func() {
		if err := e.Start(":" + s.port); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal(fmt.Sprintf("Port %s in use: server won't start", s.port))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}

func NewServer(config Config) *Server {
	database := redis.GetClient(config.REDIS_URL())

	e := echo.New()
	server := &Server{
		port: config.PORT(),
		e:    e,
	}

	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339} ${status} ${method} ${uri} ${latency_human} [${user_agent} ${remote_ip}]\n",
	}))
	e.Use(middleware.Recover())

	apiGroup := e.Group("/api")
	apiGroup.RouteNotFound("/*", func(c echo.Context) error { return c.NoContent(http.StatusNotFound) })

	todos.NewComponent(apiGroup.Group("/todos"), database)

	return server
}
