package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

type ctxKey string

const (
	ctxKeyLogger ctxKey = "logger"
)

func getLogger(r *http.Request) *slog.Logger {
	return r.Context().Value(ctxKeyLogger).(*slog.Logger)
}

func setLogger(r *http.Request, l *slog.Logger) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), ctxKeyLogger, l))
}

func newLogger(appEnv string) *slog.Logger {
	var handler slog.Handler
	if appEnv == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = tint.NewHandler(os.Stdout, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
		})
	}

	return slog.New(handler)
}
