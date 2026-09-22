package storage

import (
	"context"
	"nookbuddy/migrations"
	"testing"
)

func TestMigrate_SkipsAlreadyAppliedVersions(t *testing.T) {
	path := tempDBPath(t)
	ctx := context.Background()

	store, _ := openTestStore(t, path, migrations.FS)
	versionBefore, err := currentSchemaVersion(ctx, store.db)
	if err != nil {
		t.Fatalf("currentSchemaVersion: %v", err)
	}

	if err := applyMigrations(ctx, store.db, migrations.FS); err != nil {
		t.Fatalf("second applyMigrations: %v", err)
	}

	versionAfter, err := currentSchemaVersion(ctx, store.db)
	if err != nil {
		t.Fatalf("currentSchemaVersion: %v", err)
	}
	if versionAfter != versionBefore {
		t.Errorf("schema_version = %d, want unchanged at %d", versionAfter, versionBefore)
	}

	var cosmetics int
	row := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cosmetics`)
	if err := row.Scan(&cosmetics); err != nil {
		t.Fatalf("count cosmetics: %v", err)
	}
	if cosmetics != len(CosmeticIDs) {
		t.Errorf("cosmetics rows = %d, want %d (no duplicates)", cosmetics, len(CosmeticIDs))
	}

	var players int
	row = store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM player`)
	if err := row.Scan(&players); err != nil {
		t.Fatalf("count player: %v", err)
	}
	if players != 1 {
		t.Errorf("player rows = %d, want 1 (no duplicates)", players)
	}
}

func TestMigrationVersion(t *testing.T) {
	version, err := migrationVersion("0001_init.sql")
	if err != nil {
		t.Fatalf("migrationVersion: %v", err)
	}
	if version != 1 {
		t.Errorf("version = %d, want 1", version)
	}

	if _, err := migrationVersion("init.sql"); err == nil {
		t.Error("expected an error for a migration without a numeric prefix")
	}
	if _, err := migrationVersion("abc_init.sql"); err == nil {
		t.Error("expected an error for a migration with an invalid numeric prefix")
	}
}
