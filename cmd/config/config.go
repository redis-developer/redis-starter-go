package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Environment string

const (
	Development Environment = "development"
	Test        Environment = "test"
	Production  Environment = "production"
)

var validLogLevels = map[string]struct{}{
	"fatal":  {},
	"error":  {},
	"warn":   {},
	"info":   {},
	"debug":  {},
	"trace":  {},
	"silent": {},
}

type Config struct {
	AppEnv       Environment
	Port         int
	RedisURL     string
	LogLevel     string
	LogStreamKey string
}

func (c Config) IsDevelopment() bool {
	return c.AppEnv == Development
}

func (c Config) IsTest() bool {
	return c.AppEnv == Test
}

func (c Config) IsProduction() bool {
	return c.AppEnv == Production
}

func (c Config) Address() string {
	return ":" + strconv.Itoa(c.Port)
}

func Load() (Config, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return Config{}, fmt.Errorf("get working directory: %w", err)
	}

	fileEnv, err := loadDotEnv(workingDir)
	if err != nil {
		return Config{}, err
	}

	return parseConfig(func(key string) (string, bool) {
		if value, ok := os.LookupEnv(key); ok {
			return value, true
		}

		value, ok := fileEnv[key]
		return value, ok
	})
}

func parseConfig(lookup func(string) (string, bool)) (Config, error) {
	cfg := Config{
		AppEnv:       Development,
		Port:         8080,
		LogLevel:     "info",
		LogStreamKey: "logs",
	}

	rawEnv := getString(lookup, "APP_ENV")
	if rawEnv == "" {
		rawEnv = getString(lookup, "NODE_ENV")
	}
	rawEnv = strings.ToLower(strings.TrimSpace(rawEnv))

	if rawEnv != "" {
		switch Environment(rawEnv) {
		case Development, Test, Production:
			cfg.AppEnv = Environment(rawEnv)
		default:
			return Config{}, fmt.Errorf("APP_ENV must be one of development, test, production")
		}
	}

	if portValue := strings.TrimSpace(getString(lookup, "PORT")); portValue != "" {
		port, err := strconv.Atoi(portValue)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, fmt.Errorf("PORT must be a valid port number")
		}
		cfg.Port = port
	}

	redisURL := strings.TrimSpace(getString(lookup, "REDIS_URL"))
	if redisURL == "" {
		return Config{}, fmt.Errorf("REDIS_URL is required")
	}

	parsedURL, err := url.Parse(redisURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return Config{}, fmt.Errorf("REDIS_URL must be a valid URL")
	}
	cfg.RedisURL = redisURL

	logLevel := strings.ToLower(strings.TrimSpace(getString(lookup, "LOG_LEVEL")))
	if logLevel != "" {
		if _, ok := validLogLevels[logLevel]; !ok {
			return Config{}, fmt.Errorf("LOG_LEVEL must be one of fatal, error, warn, info, debug, trace, silent")
		}
		cfg.LogLevel = logLevel
	}

	if logStreamKey := strings.TrimSpace(getString(lookup, "LOG_STREAM_KEY")); logStreamKey != "" {
		cfg.LogStreamKey = logStreamKey
	}

	return cfg, nil
}

func getString(lookup func(string) (string, bool), key string) string {
	value, ok := lookup(key)
	if !ok {
		return ""
	}

	return value
}

func loadDotEnv(startDir string) (map[string]string, error) {
	currentDir := startDir

	for {
		candidate := filepath.Join(currentDir, ".env")
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			values, readErr := godotenv.Read(candidate)
			if readErr != nil {
				return nil, fmt.Errorf("read %s: %w", candidate, readErr)
			}

			return values, nil
		}

		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat %s: %w", candidate, err)
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return map[string]string{}, nil
		}

		currentDir = parentDir
	}
}
