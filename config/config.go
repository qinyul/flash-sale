package config

import (
	"log/slog"
	"os"
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

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		slog.Info(".env not found, relying on environment variables")
	}

	maxOpenConns, err := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "50"))
	if err != nil {
		maxOpenConns = 50
	}
	maxIdleConns, err := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "25"))
	if err != nil {
		maxIdleConns = 25
	}

	// default max request body size 1MB
	maxBodyBytes, err := strconv.Atoi(getEnv("MAX_BODY_BYTES", "1048576"))
	if err != nil {
		maxBodyBytes = 1048576
	}

	connMaxLifetime, _ := time.ParseDuration(getEnv("DB_CONN_MAX_LIFETIME", "5m"))
	connMaxIdletime, _ := time.ParseDuration(getEnv("DB_CONN_MAX_IDLE_TIME", "1m"))

	readTimeout, _ := time.ParseDuration(getEnv("APP_READ_TIMEOUT", "5s"))
	writeTimeout, _ := time.ParseDuration(getEnv("APP_WRITE_TIMEOUT", "10s"))
	idleTimeout, _ := time.ParseDuration(getEnv("APP_IDLE_TIMEOUT", "120s"))

	return &Config{
		App: AppConfig{
			Port:         getEnv("APP_PORT", "8080"),
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
			MaxBodyBytes: int64(maxBodyBytes),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "user"),
			Password:        getEnv("DB_PASS", "password"),
			DBName:          getEnv("DB_NAME", "flashsale"),
			MaxOpenConns:    maxOpenConns,
			MaxIdleConns:    maxIdleConns,
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			ConnMaxLifetime: connMaxLifetime,
			ConnMaxIdleTime: connMaxIdletime,
		},
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
