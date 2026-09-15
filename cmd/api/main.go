package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yodzafar/url-shortener-service/internal/config"
	"github.com/yodzafar/url-shortener-service/pkg/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	connCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(connCtx, cfg.DB.URL, cfg.DB.MaxConns)

	if err != nil {
		return err
	}
	defer pool.Close()

	return nil
}
