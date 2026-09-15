package logger

import (
	"log/slog"
	"os"
)

func New(level string, isLocal bool) *slog.Logger {
	var lvl slog.Level
	_ = lvl.UnmarshalText([]byte(level)) // "debug" | "info" | "warn" | "error"

	opts := &slog.HandlerOptions{Level: lvl}
	var h slog.Handler

	if isLocal {
		h = slog.NewTextHandler(os.Stdout, opts)
	} else {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}

	l := slog.New(h)

	slog.SetDefault(l)
	return l
}
