-- name: CreateRecord :exec
INSERT INTO records (id, ts, failure, description) VALUES (?, ?, ?, ?);

-- name: GetRecords :many
SELECT * FROM records;

-- name: GetRecordByLastFailure :one
SELECT * FROM records
WHERE
  failure = 1
LIMIT 1;
