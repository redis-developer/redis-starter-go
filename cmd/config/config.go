package config

import (
	"fmt"
	"os"
	"path"
	"runtime"

	"github.com/joho/godotenv"
)

type Config struct{}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func (c *Config) Port() string {
	return getEnv("PORT", "3000")
}

func (c *Config) RedisUrl() string {
	return getEnv("REDIS_URL", "redis://localhost:6379")
}

func New() *Config {
	return &Config{}
}

func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "../..")
	err := os.Chdir(dir)

	if err != nil {
		panic(err)
	}

	// Load environment variables
	err = godotenv.Load()

	if err != nil {
		fmt.Println(err)
		path, _ := os.Getwd()
		fmt.Println(path)
		fmt.Println("failed to load environment file, " +
			"assuming environment variables are already loaded")
	}
}
