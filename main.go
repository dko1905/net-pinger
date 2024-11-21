package main

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
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

const (
	SQL_INSERT_RECORD             = "INSERT INTO records (id, ts, failure, description) VALUES (?, ?, ?, ?);"
	SQL_SELECT_ALL_RECORDS        = "SELECT id, ts, failure, description FROM records;"
	SQL_SELECT_LAST_FAILED_RECORD = "SELECT id, ts, failure, description FROM records LIMIT 1;"
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

	appDb, found := os.LookupEnv("APP_DB")
	if !found {
		panic(errors.New("APP_DB undefined"))
	}

	// Logging
	logger := newLogger(appEnv)

	// Database
	db, err := sql.Open("sqlite3", appDb)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		panic(err)
	}

	dbDriver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
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

		saveRecord := func(ok bool, msg string) {
			q, err := db.Prepare(SQL_INSERT_RECORD)
			if err != nil {
				panic(err)
			}

			record_id := uuid.New()
			record_ts := time.Now().UTC().Format(time.RFC3339)
			q.Exec(record_id, record_ts, ok, msg)
		}

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
				prevFailure = true
				saveRecord(true, msg)
			} else if resp.StatusCode != http.StatusNoContent {
				// Failed, wrong response code
				msg := fmt.Sprintf("wrong status code returned from Google: %s", resp.Status)

				logger.Debug(msg)
				prevFailure = true
				saveRecord(true, msg)
			} else {
				msg := fmt.Sprintf("successfully reached Google: %s", resp.Status)

				// Success
				if prevFailure {
					saveRecord(false, msg)
					prevFailure = false
				}
			}

			time.Sleep(appInterval)
		}
	}()

	// Routes
	// -Views
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		q, err := db.Prepare(SQL_SELECT_ALL_RECORDS)
		if err != nil {
			panic(err)
		}

		rows, err := q.Query()
		if err != nil {
			panic(err)
		}

		viewData := struct {
			LastRecord *Record
			Records    []Record
		}{}

		for rows.Next() {
			var record Record
			var record_ts string
			if err := rows.Scan(&record.ID, &record_ts, &record.Failure, &record.Description); err != nil {
				panic(err)
			}
			record.TS, err = time.Parse(time.RFC3339, record_ts)
			if err != nil {
				panic(err)
			}
			viewData.Records = append(viewData.Records, record)
		}

		if len(viewData.Records) > 1 {
			viewData.LastRecord = &(viewData.Records[len(viewData.Records)-1])
		}

		if err := rows.Err(); err != nil {
			panic(err)
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
