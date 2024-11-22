package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net-pinger/db"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	slogchi "github.com/samber/slog-chi"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed templates
var templates embed.FS

var fns = template.FuncMap{
	"last": func(x int, a interface{}) bool {
		return x == reflect.ValueOf(a).Len()-1
	},
}

//go:embed migrations
var migrations embed.FS

func main() {
	// Config
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	appEnv, found := os.LookupEnv("APP_ENV")
	if !found {
		appEnv = "development"
	}

	appAddr, found := os.LookupEnv("APP_ADDR")
	if !found {
		appAddr = ":3000"
	}

	appDB, found := os.LookupEnv("APP_DB")
	if !found {
		panic(errors.New("APP_DB undefined"))
	}

	// Logging
	logger := newLogger(appEnv)

	// Database
	sqliteDB, err := sql.Open("sqlite3", appDB)
	if err != nil {
		panic(err)
	}
	defer sqliteDB.Close()

	_, err = sqliteDB.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		panic(err)
	}

	dbDriver, err := sqlite3.WithInstance(sqliteDB, &sqlite3.Config{})
	if err != nil {
		panic(err)
	}

	migrationsIofs, err := iofs.New(migrations, "migrations")
	if err != nil {
		panic(err)
	}
	m, err := migrate.NewWithInstance("iofs", migrationsIofs, "sqlite3", dbDriver)
	if err != nil {
		panic(err)
	}

	err = m.Up()
	if err == nil {
		logger.Info("Database migrated without error")
	} else if errors.Is(err, migrate.ErrNoChange) {
		logger.Info("Database has no pending migrations")
	} else {
		panic(err)
	}

	queries := db.New(sqliteDB)

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

	// App state
	go func() {
		appInterval := 1 * time.Second

		prevFailure := false // `Failure` of previous record
		client := http.Client{
			Timeout: 5 * time.Second,
		}
		for {
			resp, err := client.Get("https://google.com/generate_204")

			if err != nil {
				// Failed to fetch
				msg := fmt.Sprintf("failed to reach Google: %v", err)

				logger.Debug(msg)
				if !prevFailure {
					queries.CreateRecord(context.Background(), db.CreateRecordParams{
						ID:          uuid.NewString(),
						Ts:          time.Now().UTC().Format(time.RFC3339),
						Failure:     1,
						Description: msg,
					})
					prevFailure = true
				}
			} else if resp.StatusCode != http.StatusNoContent {
				// Failed, wrong response code
				msg := fmt.Sprintf("wrong status code returned from Google: %s", resp.Status)

				logger.Debug(msg)
				if !prevFailure {
					prevFailure = true
					queries.CreateRecord(context.Background(), db.CreateRecordParams{
						ID:          uuid.NewString(),
						Ts:          time.Now().UTC().Format(time.RFC3339),
						Failure:     1,
						Description: msg,
					})
				}
			} else {
				msg := fmt.Sprintf("successfully reached Google: %s", resp.Status)

				// Success
				if prevFailure {
					queries.CreateRecord(context.Background(), db.CreateRecordParams{
						ID:          uuid.NewString(),
						Ts:          time.Now().UTC().Format(time.RFC3339),
						Failure:     0,
						Description: msg,
					})
					prevFailure = false
				}
			}

			time.Sleep(appInterval)
		}
	}()

	// Routes
	// -Views
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		records, err := queries.GetRecords(r.Context())
		if err != nil {
			panic(err)
		}

		var lastFailureStr string
		lastFailure, err := queries.GetRecordByLastFailure(r.Context())
		if errors.Is(err, sql.ErrNoRows) {
			lastFailureStr = "No last error"
		} else if err != nil {
			panic(err)
		} else {
			lastFailureStr2, err := json.Marshal(lastFailure)
			if err != nil {
				panic(err)
			}
			lastFailureStr = string(lastFailureStr2)
		}

		viewData := struct {
			LastFailed string
			Records    []db.Record
		}{
			Records:    records,
			LastFailed: lastFailureStr,
		}
		err = tmpl.Execute(w, viewData)
		if err != nil {
			panic(err)
		}
	})

	// -APIs
	api := chi.NewRouter()

	api.Get("/records", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	r.Mount("/api", api)

	// Startup
	logger.Info("NetPinger listening on", "addr", appAddr, "env", appEnv)
	http.ListenAndServe(appAddr, r)
}
