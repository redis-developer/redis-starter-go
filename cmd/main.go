package main

import (
	"context"
	"log"

	"github.com/redis-developer/redis-starter-go/cmd/config"
	server "github.com/redis-developer/redis-starter-go/pkg"
	"github.com/redis-developer/redis-starter-go/pkg/components/todos"
	"github.com/redis-developer/redis-starter-go/pkg/logging"
	redispkg "github.com/redis-developer/redis-starter-go/pkg/redis"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	redisClient, err := redispkg.NewClient(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatal(err)
	}

	logger, closeLogger, err := logging.New(logging.Options{
		Environment: string(cfg.AppEnv),
		Level:       cfg.LogLevel,
		StreamKey:   cfg.LogStreamKey,
		RedisClient: redisClient,
	})
	if err != nil {
		log.Fatal(err)
	}

	model := todos.NewModel(redisClient)
	if err := model.CreateIndexIfNotExists(ctx); err != nil {
		log.Fatal(err)
	}

	srv := server.New(
		cfg,
		logger,
		model,
		closeLogger,
		func(context.Context) error {
			return redisClient.Close()
		},
	)

	logger.Info("redis go starter server listening", "port", cfg.Port)
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
