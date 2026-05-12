package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aakash1408/shortener/internal/config"
	"github.com/aakash1408/shortener/internal/server"
	"github.com/aakash1408/shortener/internal/store"
)

func main() {
	os.Exit(run())
}

func run() int {
	// 1. load config — fail fast if anything missing
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return 1
	}

	// 2. set up logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	}))

	// 3. connect to database
	ctx := context.Background()
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		return 1
	}
	logger.Info("connected to database")

	// 4. run migrations
	if err := st.RunMigrations(ctx); err != nil {
		logger.Error("failed to run migrations", "error", err)
		return 1
	}
	logger.Info("migrations complete")

	// 5. create server
	srv := server.New(cfg, st, logger)

	// 6. listen for Ctrl+C or SIGTERM
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 7. start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Start()
	}()

	// 8. wait for interrupt or server error
	select {
	case <-sigCtx.Done():
		logger.Info("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown error", "error", err)
			return 1
		}
	case err := <-serverErr:
		if err != nil {
			logger.Error("server error", "error", err)
			return 1
		}
	}

	logger.Info("server stopped")
	return 0
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
