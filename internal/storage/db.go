package storage

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"nookbuddy/migrations"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const corruptSuffixLayout = "20060102-150405"

type Store struct {
	db *sql.DB
}

func DefaultPath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "NookBuddy", "nookbuddy.db"), nil
}

func Open(ctx context.Context, path string) (*Store, Notice, error) {
	return openWithMigrations(ctx, path, migrations.FS)
}

func openWithMigrations(ctx context.Context, path string, fsys fs.FS) (*Store, Notice, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, NoticeNone, err
	}

	store, err := openMigrated(ctx, path, fsys)
	if err == nil {
		return store, NoticeNone, nil
	}

	slog.Warn("nookbuddy: database unusable, starting fresh", "path", path, "error", err)

	if renameErr := renameAside(path); renameErr != nil {
		return nil, NoticeNone, fmt.Errorf("could not set the unusable database aside: %w", renameErr)
	}

	store, err = openMigrated(ctx, path, fsys)
	if err != nil {
		return nil, NoticeNone, fmt.Errorf("could not create a fresh database: %w", err)
	}

	return store, NoticeRecoveredFromCorruption, nil
}

func openMigrated(ctx context.Context, path string, fsys fs.FS) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := applyMigrations(ctx, db, fsys); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func renameAside(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	aside := fmt.Sprintf("%s.corrupt-%s", path, time.Now().UTC().Format(corruptSuffixLayout))
	return os.Rename(path, aside)
}
