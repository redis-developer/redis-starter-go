package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis-developer/redis-starter-go/cmd/config"
	"github.com/redis-developer/redis-starter-go/pkg/components/todos"
)

const shutdownTimeout = 10 * time.Second

type Server struct {
	address   string
	e         *echo.Echo
	shutdowns []func(context.Context) error
}

type errorResponse struct {
	Message string `json:"message"`
}

func newHTTPErrorHandler(logger *slog.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		status := http.StatusInternalServerError
		message := "internal server error"

		var httpError *echo.HTTPError
		if errors.As(err, &httpError) {
			status = httpError.Code
			message = fmt.Sprint(httpError.Message)
		} else {
			logger.Error("unhandled error", slog.Any("error", err))
		}

		if status >= http.StatusInternalServerError && httpError != nil {
			logger.Error("request failed", slog.Any("error", err))
		}

		if jsonErr := c.JSON(status, errorResponse{Message: message}); jsonErr != nil {
			logger.Error("write error response", slog.Any("error", jsonErr))
		}
	}
}

func requestLogger(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			startedAt := time.Now()
			err := next(c)
			if err != nil {
				c.Error(err)
			}

			attrs := []any{
				slog.String("method", c.Request().Method),
				slog.String("path", c.Request().URL.Path),
				slog.Int("status", c.Response().Status),
				slog.String("latency", time.Since(startedAt).String()),
				slog.String("remote_ip", c.RealIP()),
				slog.String("user_agent", c.Request().UserAgent()),
			}
			if err != nil {
				attrs = append(attrs, slog.String("error", err.Error()))
			}

			logger.Info("request completed", attrs...)
			return nil
		}
	}
}

func setupAPI(logger *slog.Logger, model todos.Model) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(requestLogger(logger.With("component", "http")))
	e.Use(middleware.Recover())
	e.HTTPErrorHandler = newHTTPErrorHandler(logger)

	apiGroup := e.Group("/api")
	apiGroup.RouteNotFound("/*", func(c echo.Context) error {
		return c.NoContent(http.StatusNotFound)
	})

	controller := todos.NewController(model, logger)
	todos.NewRouter(apiGroup.Group("/todos"), controller)
	return e
}

func (s *Server) Handler() http.Handler {
	return s.e
}

func (s *Server) Shutdown(ctx context.Context) error {
	var shutdownErr error

	if err := s.e.Shutdown(ctx); err != nil {
		shutdownErr = err
	}

	for _, shutdown := range s.shutdowns {
		if err := shutdown(ctx); err != nil && shutdownErr == nil {
			shutdownErr = err
		}
	}

	return shutdownErr
}

func (s *Server) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		if err := s.e.Start(s.address); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		_ = s.Shutdown(shutdownCtx)
		return fmt.Errorf("start server: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return s.Shutdown(shutdownCtx)
	}
}

func New(cfg config.Config, logger *slog.Logger, model todos.Model, shutdowns ...func(context.Context) error) *Server {
	return &Server{
		address:   ":" + strconv.Itoa(cfg.Port),
		e:         setupAPI(logger, model),
		shutdowns: shutdowns,
	}
}
