package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/yodzafar/url-shortener-service/internal/app"
	"github.com/yodzafar/url-shortener-service/internal/config"
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

	a, cleanup, err := app.InitApp(ctx, cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	return a.Run(ctx)
}
