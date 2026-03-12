package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfigDefaults(t *testing.T) {
	cfg, err := parseConfig(func(key string) (string, bool) {
		if key == "REDIS_URL" {
			return "redis://localhost:6379", true
		}

		return "", false
	})

	require.NoError(t, err)
	assert.Equal(t, Development, cfg.AppEnv)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, "redis://localhost:6379", cfg.RedisURL)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "logs", cfg.LogStreamKey)
}

func TestParseConfigNodeEnvFallback(t *testing.T) {
	cfg, err := parseConfig(func(key string) (string, bool) {
		values := map[string]string{
			"NODE_ENV":  "test",
			"REDIS_URL": "redis://localhost:6379",
		}

		value, ok := values[key]
		return value, ok
	})

	require.NoError(t, err)
	assert.Equal(t, Test, cfg.AppEnv)
	assert.True(t, cfg.IsTest())
}

func TestParseConfigRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		values  map[string]string
		message string
	}{
		{
			name: "missing redis url",
			values: map[string]string{
				"PORT": "8080",
			},
			message: "REDIS_URL is required",
		},
		{
			name: "invalid env",
			values: map[string]string{
				"APP_ENV":   "staging",
				"REDIS_URL": "redis://localhost:6379",
			},
			message: "APP_ENV must be one of development, test, production",
		},
		{
			name: "invalid port",
			values: map[string]string{
				"PORT":      "nope",
				"REDIS_URL": "redis://localhost:6379",
			},
			message: "PORT must be a valid port number",
		},
		{
			name: "invalid redis url",
			values: map[string]string{
				"REDIS_URL": "localhost",
			},
			message: "REDIS_URL must be a valid URL",
		},
		{
			name: "invalid log level",
			values: map[string]string{
				"REDIS_URL": "redis://localhost:6379",
				"LOG_LEVEL": "verbose",
			},
			message: "LOG_LEVEL must be one of fatal, error, warn, info, debug, trace, silent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseConfig(func(key string) (string, bool) {
				value, ok := tt.values[key]
				return value, ok
			})

			require.EqualError(t, err, tt.message)
		})
	}
}

func TestLoadDotEnvFindsParentEnvFile(t *testing.T) {
	rootDir := t.TempDir()
	childDir := filepath.Join(rootDir, "a", "b", "c")

	require.NoError(t, os.MkdirAll(childDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, ".env"), []byte("REDIS_URL=redis://root:6379\n"), 0o644))

	values, err := loadDotEnv(childDir)

	require.NoError(t, err)
	assert.Equal(t, "redis://root:6379", values["REDIS_URL"])
}
