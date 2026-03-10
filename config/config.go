package config

import (
	"fmt"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
}

type AppConfig struct {
	Port         string
	MaxBodyBytes int64
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
}

// EnvGetter defines the signature for any function that can look up an environment variable
// it will matches the exact signature of os.Lookupenv
type EnvGetter func(key string) (string, bool)

func LoadConfig(getEnvFn EnvGetter) (*Config, error) {
	_ = godotenv.Load()
	// default max request body size 1MB
	maxOpenConns, err := parseEnvInt(getEnvFn, "DB_MAX_OPEN_CONNS", 100)
	if err != nil {
		return nil, err
	}

	maxIdleConns, err := parseEnvInt(getEnvFn, "DB_MAX_IDLE_CONNS", 100)
	if err != nil {
		return nil, err
	}

	maxBodyBytes, err := parseEnvInt(getEnvFn, "MAX_BODY_BYTES", 1048576)
	if err != nil {
		return nil, err
	}

	readTimeout, err := parseEnvDuration(getEnvFn, "APP_READ_TIMEOUT", "5s")
	if err != nil {
		return nil, err
	}

	writeTimeout, err := parseEnvDuration(getEnvFn, "APP_WRITE_TIMEOUT", "10s")
	if err != nil {
		return nil, err
	}

	idleTimeout, err := parseEnvDuration(getEnvFn, "APP_IDLE_TIMEOUT", "120s")
	if err != nil {
		return nil, err
	}

	connMaxLifetime, err := parseEnvDuration(getEnvFn, "DB_CONN_MAX_LIFETIME", "30m")
	if err != nil {
		return nil, err
	}

	connMaxIdleTime, err := parseEnvDuration(getEnvFn, "DB_CONN_MAX_IDLE_TIME", "5m")
	if err != nil {
		return nil, err
	}

	return &Config{
		App: AppConfig{
			Port:         getEnvString(getEnvFn, "APP_PORT", "8080"),
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
			MaxBodyBytes: int64(maxBodyBytes),
		},
		Database: DatabaseConfig{
			Host:            getEnvString(getEnvFn, "DB_HOST", "localhost"),
			Port:            getEnvString(getEnvFn, "DB_PORT", "5432"),
			User:            getEnvString(getEnvFn, "DB_USER", "user"),
			Password:        getEnvString(getEnvFn, "DB_PASS", "password"),
			DBName:          getEnvString(getEnvFn, "DB_NAME", "flashsale"),
			MaxOpenConns:    maxOpenConns,
			MaxIdleConns:    maxIdleConns,
			SSLMode:         getEnvString(getEnvFn, "DB_SSL_MODE", "disable"),
			ConnMaxLifetime: connMaxLifetime,
			ConnMaxIdleTime: connMaxIdleTime,
		},
	}, nil
}

func getEnvString(getEnvFn EnvGetter, key, fallback string) string {
	if value, exists := getEnvFn(key); exists {
		return value
	}
	return fallback
}

func parseEnvInt(getEnvFn EnvGetter, key string, fallback int) (int, error) {
	valStr := getEnvString(getEnvFn, key, "")
	if valStr == "" {
		return fallback, nil
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, fmt.Errorf("invalid integer for %s: %w", key, err)
	}
	return val, nil
}

func parseEnvDuration(getEnvFn EnvGetter, key, fallback string) (time.Duration, error) {
	valStr := getEnvString(getEnvFn, key, fallback)
	val, err := time.ParseDuration(valStr)
	if err != nil {
		return 0, fmt.Errorf("Invalid duration for :%s: %w", key, err)
	}
	return val, nil
}
