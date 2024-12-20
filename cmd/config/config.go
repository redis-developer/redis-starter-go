package config

import "os"

type Config struct{}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func (c *Config) REDIS_URL() string {
	return getEnv("REDIS_URL", "redis://localhost:6379")
}

func (c *Config) PORT() string {
	return getEnv("PORT", "3000")
}
