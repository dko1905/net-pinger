package main

import (
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	_ "embed"

	slogchi "github.com/samber/slog-chi"
)

//go:embed templates
var templates embed.FS

func main() {
	// Config
	appEnv, found := os.LookupEnv("APP_ENV")
	if !found {
		appEnv = "development"
	}

	appAddr, found := os.LookupEnv("APP_ADDR")
	if !found {
		appAddr = ":3000"
	}

	// Logging
	logger := newLogger(appEnv)

	// Chi
	r := chi.NewRouter()

	r.Use(slogchi.NewWithConfig(logger, slogchi.Config{
		DefaultLevel:     slog.LevelDebug,
		ClientErrorLevel: slog.LevelWarn,
		ServerErrorLevel: slog.LevelError,
	}))
	r.Use(middleware.Recoverer)

	// Templating
	tmpl, err := template.ParseFS(templates, "templates/*.html")
	if err != nil {
		panic(err)
	}

	// Routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, nil)
	})

	// Startup
	logger.Info("NetPinger listening on", "addr", appAddr, "env", appEnv)
	http.ListenAndServe(appAddr, r)
}
