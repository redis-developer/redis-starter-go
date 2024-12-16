package redis

import (
	"github.com/redis/go-redis/v9"
)

func GetClient(url string) *redis.Client {
	opt, err := redis.ParseURL(url)

  if err != nil {
		panic(err)
	}

	client := redis.NewClient(opt)

  return client
}
