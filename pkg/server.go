package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"
)

type Config interface {
	Port() string
	RedisUrl() string
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

	// Wait for interrupt signal to gracefully shutdown the server with a timeout
	// of 10 seconds.
	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}

func New(config Config) *Server {
  e := SetupApp(config)
	server := &Server{
		port: config.Port(),
		e:    e,
	}

  return server
}
