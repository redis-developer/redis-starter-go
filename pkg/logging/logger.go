package logging

import (
	"context"
	"io"
	"log/slog"
	"os"

	redis "github.com/redis/go-redis/v9"
)

type Options struct {
	Environment string
	Level       string
	StreamKey   string
	RedisClient *redis.Client
	Output      io.Writer
}

func New(opts Options) (*slog.Logger, func(context.Context) error, error) {
	output := opts.Output
	if output == nil {
		output = os.Stdout
	}

	if opts.Level == "silent" {
		return slog.New(NopHandler{}), noopClose, nil
	}

	level := parseLevel(opts.Level)
	handlerOptions := &slog.HandlerOptions{Level: level}

	var handlers []slog.Handler
	if opts.Environment == "production" {
		handlers = append(handlers, slog.NewJSONHandler(output, handlerOptions))
	} else {
		handlers = append(handlers, slog.NewTextHandler(output, handlerOptions))
	}

	closeFunc := noopClose
	if opts.RedisClient != nil {
		redisHandler := NewRedisHandler(level, opts.RedisClient, opts.StreamKey)
		handlers = append(handlers, redisHandler)
		closeFunc = redisHandler.Close
	}

	return slog.New(FanoutHandler{Handlers: handlers}), closeFunc, nil
}

func parseLevel(level string) slog.Level {
	switch level {
	case "trace", "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error", "fatal":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func noopClose(context.Context) error {
	return nil
}

type NopHandler struct{}

func (NopHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (NopHandler) Handle(context.Context, slog.Record) error { return nil }
func (NopHandler) WithAttrs([]slog.Attr) slog.Handler        { return NopHandler{} }
func (NopHandler) WithGroup(string) slog.Handler             { return NopHandler{} }

type FanoutHandler struct {
	Handlers []slog.Handler
}

func (h FanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.Handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}

	return false
}

func (h FanoutHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, handler := range h.Handlers {
		if !handler.Enabled(ctx, record.Level) {
			continue
		}

		if err := handler.Handle(ctx, record.Clone()); err != nil {
			return err
		}
	}

	return nil
}

func (h FanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, 0, len(h.Handlers))

	for _, handler := range h.Handlers {
		next = append(next, handler.WithAttrs(attrs))
	}

	return FanoutHandler{Handlers: next}
}

func (h FanoutHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, 0, len(h.Handlers))

	for _, handler := range h.Handlers {
		next = append(next, handler.WithGroup(name))
	}

	return FanoutHandler{Handlers: next}
}
