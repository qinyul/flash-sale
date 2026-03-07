package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/qinyul/flash-sale/config"
	"github.com/qinyul/flash-sale/handler"
	"github.com/qinyul/flash-sale/infrastructure"
	"github.com/qinyul/flash-sale/repository"
	"github.com/qinyul/flash-sale/service"
)

func main() {
	// init configuration
	cfg := config.LoadConfig()

	// setup infrastucture (Database)
	db, err := infrastructure.NewPostgresDB(cfg.Database)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.DB.Close()

	// Start DB Monitoring in background
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		db.MonitorDBStats(ctx)
	}()

	// Init domain layer
	repo := repository.NewFlashSaleRepository(db.DB)
	svc := service.NewFlashSaleService(repo)
	hdl := handler.NewFlashSaleHandler(svc, cfg.App.MaxBodyBytes)

	// setup routing
	mux := http.NewServeMux()
	hdl.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      mux,
		ReadTimeout:  cfg.App.ReadTimeout,
		WriteTimeout: cfg.App.WriteTimeout,
		IdleTimeout:  cfg.App.IdleTimeout,
	}

	go func() {
		slog.Info("Server starting", "port", cfg.App.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed to start", "error", err)
			stop()
		}
	}()

	// Graceful shutdown
	<-ctx.Done()
	slog.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}
	wg.Wait()

	slog.Info("All system is offline")
}
