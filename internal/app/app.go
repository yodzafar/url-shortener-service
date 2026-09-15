package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

type App struct {
	Server *http.Server
	Log    *slog.Logger
}

func NewApp(server *http.Server, log *slog.Logger) *App {
	return &App{Server: server, Log: log}
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		a.Log.Info("listening", "addr", a.Server.Addr)
		if err := a.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return a.Server.Shutdown(shutdownCtx)
}
