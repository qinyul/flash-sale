package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig_DefaultValues(t *testing.T) {
	emptyEnvGetter := func(key string) (string, bool) {
		return "", false
	}
	cfg, err := LoadConfig(emptyEnvGetter)

	require.NoError(t, err)
	require.Equal(t, "8080", cfg.App.Port)
	require.Equal(t, "localhost", cfg.Database.Host)

	require.Equal(t, 100, cfg.Database.MaxOpenConns)
	require.Equal(t, 30*time.Minute, cfg.Database.ConnMaxLifetime)
}

func TestLoadConfig_WithEnvOverrides(t *testing.T) {
	mockEnv := map[string]string{
		"APP_PORT":             "9090",
		"DB_HOST":              "db-prod",
		"DB_MAX_OPEN_CONNS":    "200",
		"DB_CONN_MAX_LIFETIME": "10m",
	}

	mockEnvGetter := func(key string) (string, bool) {
		val, exist := mockEnv[key]
		return val, exist
	}
	cfg, err := LoadConfig(mockEnvGetter)

	require.NoError(t, err)
	require.Equal(t, "9090", cfg.App.Port)
	require.Equal(t, "db-prod", cfg.Database.Host)
	require.Equal(t, 200, cfg.Database.MaxOpenConns)
	require.Equal(t, 10*time.Minute, cfg.Database.ConnMaxLifetime)
}

func TestLoadConfig_InvalidIntReturnsError(t *testing.T) {
	badEnvGetter := func(key string) (string, bool) {
		if key == "DB_MAX_OPEN_CONNS" {
			return "garbage_data", true
		}
		return "", false
	}
	cfg, err := LoadConfig(badEnvGetter)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid integer for DB_MAX_OPEN_CONNS")
	require.Nil(t, cfg)
}
