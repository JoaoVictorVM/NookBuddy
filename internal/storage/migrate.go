package storage

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

func currentSchemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	var exists int
	row := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_version'`)
	if err := row.Scan(&exists); err != nil {
		return 0, err
	}
	if exists == 0 {
		return 0, nil
	}

	var version int
	row = db.QueryRowContext(ctx, `SELECT version FROM schema_version LIMIT 1`)
	if err := row.Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return version, nil
}

func migrationVersion(name string) (int, error) {
	prefix, _, found := strings.Cut(name, "_")
	if !found {
		return 0, fmt.Errorf("migration %q has no numeric prefix", name)
	}
	version, err := strconv.Atoi(prefix)
	if err != nil {
		return 0, fmt.Errorf("migration %q has an invalid numeric prefix: %w", name, err)
	}
	return version, nil
}

func applyMigrations(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)

	applied, err := currentSchemaVersion(ctx, db)
	if err != nil {
		return err
	}

	for _, name := range names {
		version, err := migrationVersion(name)
		if err != nil {
			return err
		}
		if version <= applied {
			continue
		}

		statements, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		if err := applyMigration(ctx, db, string(statements), version); err != nil {
			return fmt.Errorf("migration %q failed: %w", name, err)
		}
	}

	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, statements string, version int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, statements); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE schema_version SET version = ?`, version); err != nil {
		return err
	}

	return tx.Commit()
}
