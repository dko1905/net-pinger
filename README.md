# NetPinger

Simple application to ping the internet indefinably while logging errors.

## Dev

This project uses nix to install go development tools.

```sh
# Enter nix development environment
$ nix develop
$ go run main.go logger.go
```

## Database

Generate go-bindings:
`sqlc generate`

Create migration:
`migrate create -ext sql -dir ./migrations -seq create_content_table`

Apply migration manually:
`migrate -path ./migrations -database "sqlite3:$DB_PATH" up`

In case of `Dirty database`-errors:

```sh
migrate -path ./migrations -database "sqlite3://$DB_PATH" up
# error: Dirty database version 2. Fix and force version.

migrate -path ./migrations -database "sqlite3://$DB_PATH" force 2
# No output
```
