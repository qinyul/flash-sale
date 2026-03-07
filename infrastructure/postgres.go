package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/lib/pq"
	"github.com/qinyul/flash-sale/config"
)

type PostgreDB struct {
	DB *sql.DB
}

func NewPostgresDB(cfg config.DatabaseConfig) (*PostgreDB, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)

	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	slog.Info("Postgres pool established")

	postgreDB := PostgreDB{
		DB: db,
	}
	return &postgreDB, nil
}

func (db *PostgreDB) MonitorDBStats(ctx context.Context) {
	// run DB stats every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := db.DB.Stats()

			slog.Info("DB - Stats",
				"Open", stats.OpenConnections,
				"InUse", stats.InUse,
				"Idle", stats.Idle,
				"WaitCount", stats.WaitCount,
			)

			if stats.WaitCount > 1000 {
				slog.Info("Database pool is starving! consider increasing MaxOpenConns")
			}
		}

	}
}
