package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig_DefaultValues(t *testing.T) {
	cfg := LoadConfig()

	require.Equal(t, "8080", cfg.App.Port)
	require.Equal(t, "localhost", cfg.Database.Host)
	require.Equal(t, "5432", cfg.Database.Port)
	require.Equal(t, "user", cfg.Database.User)
	require.Equal(t, "password", cfg.Database.Password)
	require.Equal(t, "flashsale", cfg.Database.DBName)
	require.Equal(t, "disable", cfg.Database.SSLMode)

	require.Equal(t, 50, cfg.Database.MaxOpenConns)
	require.Equal(t, 25, cfg.Database.MaxIdleConns)

	require.Equal(t, 5*time.Minute, cfg.Database.ConnMaxLifetime)
	require.Equal(t, 1*time.Minute, cfg.Database.ConnMaxIdleTime)
}

func TestLoadConfig_WithEnvOverrides(t *testing.T) {
	t.Setenv("APP_PORT", "9090")
	t.Setenv("DB_HOST", "db-prod")
	t.Setenv("DB_PORT", "9999")
	t.Setenv("DB_USER", "admin")
	t.Setenv("DB_PASS", "secret")
	t.Setenv("DB_NAME", "prod_db")
	t.Setenv("DB_SSL_MODE", "require")

	t.Setenv("DB_MAX_OPEN_CONNS", "100")
	t.Setenv("DB_MAX_IDLE_CONNS", "80")

	t.Setenv("DB_CONN_MAX_LIFETIME", "10m")
	t.Setenv("DB_CONN_MAX_IDLE_TIME", "2m")

	cfg := LoadConfig()

	require.Equal(t, "9090", cfg.App.Port)
	require.Equal(t, "db-prod", cfg.Database.Host)
	require.Equal(t, "9999", cfg.Database.Port)
	require.Equal(t, "admin", cfg.Database.User)
	require.Equal(t, "secret", cfg.Database.Password)
	require.Equal(t, "prod_db", cfg.Database.DBName)
	require.Equal(t, "require", cfg.Database.SSLMode)

	require.Equal(t, 100, cfg.Database.MaxOpenConns)
	require.Equal(t, 80, cfg.Database.MaxIdleConns)

	require.Equal(t, 10*time.Minute, cfg.Database.ConnMaxLifetime)
	require.Equal(t, 2*time.Minute, cfg.Database.ConnMaxIdleTime)
}

func TestLoadConfig_InvalidIntFallsBackToDefault(t *testing.T) {
	t.Setenv("DB_MAX_OPEN_CONNS", "invalid")
	t.Setenv("DB_MAX_IDLE_CONNS", "invalid")

	cfg := LoadConfig()

	require.Equal(t, 50, cfg.Database.MaxOpenConns)
	require.Equal(t, 25, cfg.Database.MaxIdleConns)
}

func TestLoadConfig_InvalidDurationFallsBackToZero(t *testing.T) {
	t.Setenv("DB_CONN_MAX_LIFETIME", "invalid")
	t.Setenv("DB_CONN_MAX_IDLE_TIME", "invalid")

	cfg := LoadConfig()

	// Because you ignored error, invalid duration becomes zero value
	require.Equal(t, time.Duration(0), cfg.Database.ConnMaxLifetime)
	require.Equal(t, time.Duration(0), cfg.Database.ConnMaxIdleTime)
}
