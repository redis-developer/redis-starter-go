package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	redis "github.com/redis/go-redis/v9"
)

const handlerBufferSize = 256

type redisEntry struct {
	level     string
	component string
	message   string
	timestamp string
	metadata  string
}

type RedisHandler struct {
	client    *redis.Client
	streamKey string
	level     slog.Level
	attrs     []slog.Attr
	groups    []string
	entries   chan redisEntry
	wg        *sync.WaitGroup
	closeOnce *sync.Once
	done      chan struct{}
}

func NewRedisHandler(level slog.Level, client *redis.Client, streamKey string) *RedisHandler {
	handler := &RedisHandler{
		client:    client,
		streamKey: streamKey,
		level:     level,
		entries:   make(chan redisEntry, handlerBufferSize),
		wg:        &sync.WaitGroup{},
		closeOnce: &sync.Once{},
		done:      make(chan struct{}),
	}

	handler.wg.Add(1)
	go handler.run()

	return handler
}

func (h *RedisHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *RedisHandler) Handle(_ context.Context, record slog.Record) error {
	entry, err := h.buildEntry(record)
	if err != nil {
		return nil
	}

	select {
	case h.entries <- entry:
	default:
	}

	return nil
}

func (h *RedisHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	next.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &next
}

func (h *RedisHandler) WithGroup(name string) slog.Handler {
	next := *h
	next.groups = append(append([]string{}, h.groups...), name)
	return &next
}

func (h *RedisHandler) Close(ctx context.Context) error {
	h.closeOnce.Do(func() {
		close(h.entries)
	})

	select {
	case <-h.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *RedisHandler) run() {
	defer h.wg.Done()
	defer close(h.done)

	for entry := range h.entries {
		_ = h.client.XAdd(context.Background(), &redis.XAddArgs{
			Stream: h.streamKey,
			ID:     "*",
			Values: map[string]any{
				"level":     entry.level,
				"component": entry.component,
				"msg":       entry.message,
				"time":      entry.timestamp,
				"metadata":  entry.metadata,
			},
		}).Err()
	}
}

func (h *RedisHandler) buildEntry(record slog.Record) (redisEntry, error) {
	metadata := map[string]any{}
	appendAttrs(metadata, h.attrs, h.groups)

	record.Attrs(func(attr slog.Attr) bool {
		appendAttrs(metadata, []slog.Attr{attr}, h.groups)
		return true
	})

	component := "root"
	if value, ok := metadata["component"]; ok {
		component = fmt.Sprint(value)
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return redisEntry{}, err
	}

	timestamp := record.Time
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	return redisEntry{
		level:     levelLabel(record.Level),
		component: component,
		message:   record.Message,
		timestamp: timestamp.UTC().Format(time.RFC3339Nano),
		metadata:  string(metadataJSON),
	}, nil
}

func appendAttrs(target map[string]any, attrs []slog.Attr, groups []string) {
	for _, attr := range attrs {
		attr.Value = attr.Value.Resolve()
		if attr.Equal(slog.Attr{}) {
			continue
		}

		if attr.Value.Kind() == slog.KindGroup {
			appendAttrs(target, attr.Value.Group(), append(groups, attr.Key))
			continue
		}

		key := attr.Key
		if len(groups) > 0 {
			key = strings.Join(append(append([]string{}, groups...), attr.Key), ".")
		}

		target[key] = attrValue(attr.Value)
	}
}

func attrValue(value slog.Value) any {
	switch value.Kind() {
	case slog.KindBool:
		return value.Bool()
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindFloat64:
		return value.Float64()
	case slog.KindInt64:
		return value.Int64()
	case slog.KindString:
		return value.String()
	case slog.KindTime:
		return value.Time().UTC().Format(time.RFC3339Nano)
	case slog.KindUint64:
		return value.Uint64()
	default:
		return value.Any()
	}
}

func levelLabel(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "error"
	case level >= slog.LevelWarn:
		return "warn"
	case level >= slog.LevelInfo:
		return "info"
	default:
		return "debug"
	}
}
