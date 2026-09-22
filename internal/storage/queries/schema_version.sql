-- name: GetSchemaVersion :one
SELECT version FROM schema_version LIMIT 1;

-- name: SetSchemaVersion :exec
UPDATE schema_version SET version = ?;
