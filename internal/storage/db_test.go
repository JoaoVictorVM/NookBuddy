package storage

import (
	"context"
	"database/sql"
	"io/fs"
	"nookbuddy/migrations"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func tempDBPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test.db")
}

func openTestStore(t *testing.T, path string, fsys fs.FS) (*Store, Notice) {
	t.Helper()
	store, notice, err := openWithMigrations(context.Background(), path, fsys)
	if err != nil {
		t.Fatalf("openWithMigrations: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, notice
}

func corruptFilesNextTo(t *testing.T, path string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	base := filepath.Base(path) + ".corrupt-"
	var found []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), base) {
			found = append(found, entry.Name())
		}
	}
	return found
}

func TestOpen_CreatesFreshDatabase(t *testing.T) {
	path := tempDBPath(t)
	store, notice := openTestStore(t, path, migrations.FS)

	if notice != NoticeNone {
		t.Errorf("notice = %v, want NoticeNone", notice)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("database file was not created: %v", err)
	}

	state, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := PlayerState{OwnedCosmetics: []string{}}
	if state.ClicksProgress != want.ClicksProgress || state.KeysProgress != want.KeysProgress ||
		state.ProjectsReady != want.ProjectsReady || state.Gold != want.Gold ||
		state.UpgradeClicksLevel != 0 || state.UpgradeKeysLevel != 0 || state.UpgradeValueLevel != 0 {
		t.Errorf("fresh state = %+v, want all zeros", state)
	}
	if len(state.OwnedCosmetics) != 0 {
		t.Errorf("OwnedCosmetics = %v, want empty", state.OwnedCosmetics)
	}
}

func TestOpen_AppliesMigrationsOnExistingDatabase(t *testing.T) {
	path := tempDBPath(t)
	ctx := context.Background()

	initial, err := fs.ReadFile(migrations.FS, "0001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	onlyFirst := fstest.MapFS{"0001_init.sql": {Data: initial}}

	store, _ := openTestStore(t, path, onlyFirst)
	if _, err := store.Save(ctx, PlayerState{Gold: 42, OwnedCosmetics: []string{"flower"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	withSecond := fstest.MapFS{
		"0001_init.sql": {Data: initial},
		"0002_note.sql": {Data: []byte(`ALTER TABLE player ADD COLUMN note TEXT;`)},
	}
	reopened, notice := openTestStore(t, path, withSecond)
	if notice != NoticeNone {
		t.Fatalf("notice = %v, want NoticeNone", notice)
	}

	version, err := currentSchemaVersion(ctx, reopened.db)
	if err != nil {
		t.Fatalf("currentSchemaVersion: %v", err)
	}
	if version != 2 {
		t.Errorf("schema_version = %d, want 2", version)
	}

	state, err := reopened.Load(ctx)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if state.Gold != 42 {
		t.Errorf("Gold = %d, want 42 (pre-existing data must survive)", state.Gold)
	}
	if len(state.OwnedCosmetics) != 1 || state.OwnedCosmetics[0] != "flower" {
		t.Errorf("OwnedCosmetics = %v, want [flower]", state.OwnedCosmetics)
	}
}

func TestOpen_RecoversFromCorruptFile(t *testing.T) {
	path := tempDBPath(t)
	if err := os.WriteFile(path, []byte("this is definitely not a sqlite database"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store, notice := openTestStore(t, path, migrations.FS)
	if notice != NoticeRecoveredFromCorruption {
		t.Errorf("notice = %v, want NoticeRecoveredFromCorruption", notice)
	}
	if found := corruptFilesNextTo(t, path); len(found) != 1 {
		t.Errorf("corrupt files = %v, want exactly one", found)
	}

	state, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load on fresh database: %v", err)
	}
	if state.Gold != 0 {
		t.Errorf("Gold = %d, want 0 on a fresh database", state.Gold)
	}
}

func TestOpen_RecoversFromFailedMigration(t *testing.T) {
	path := tempDBPath(t)
	ctx := context.Background()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	seed, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	_, err = seed.ExecContext(ctx, `
		CREATE TABLE player (id INTEGER PRIMARY KEY, note TEXT);
		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version (version) VALUES (1);
	`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("Close seed: %v", err)
	}

	initial, err := fs.ReadFile(migrations.FS, "0001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	fsys := fstest.MapFS{
		"0001_init.sql": {Data: initial},
		"0002_note.sql": {Data: []byte(`ALTER TABLE player ADD COLUMN note TEXT;`)},
	}

	store, notice := openTestStore(t, path, fsys)
	if notice != NoticeRecoveredFromCorruption {
		t.Errorf("notice = %v, want NoticeRecoveredFromCorruption", notice)
	}
	if found := corruptFilesNextTo(t, path); len(found) != 1 {
		t.Errorf("corrupt files = %v, want exactly one", found)
	}

	version, err := currentSchemaVersion(ctx, store.db)
	if err != nil {
		t.Fatalf("currentSchemaVersion: %v", err)
	}
	if version != 2 {
		t.Errorf("schema_version = %d, want 2 on the fresh database", version)
	}
}

func TestDefaultPath(t *testing.T) {
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if filepath.Base(path) != "nookbuddy.db" {
		t.Errorf("base = %q, want nookbuddy.db", filepath.Base(path))
	}
	if filepath.Base(filepath.Dir(path)) != "NookBuddy" {
		t.Errorf("parent = %q, want NookBuddy", filepath.Base(filepath.Dir(path)))
	}
}

func TestOpen_FreshDatabaseIsAtSchemaVersionOne(t *testing.T) {
	store, _ := openTestStore(t, tempDBPath(t), migrations.FS)

	version, err := currentSchemaVersion(context.Background(), store.db)
	if err != nil {
		t.Fatalf("currentSchemaVersion: %v", err)
	}
	if version != 1 {
		t.Errorf("schema_version = %d, want 1", version)
	}
}

func TestStore_SaveOnReadOnlyDatabaseReportsFailure(t *testing.T) {
	path := tempDBPath(t)
	ctx := context.Background()

	store, _ := openTestStore(t, path, migrations.FS)
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := os.Chmod(path, 0o444); err != nil {
		t.Skipf("cannot make the database read-only on this platform: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	reopened, _, err := openWithMigrations(ctx, path, migrations.FS)
	if err != nil {
		t.Skipf("read-only database could not be opened for the test: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })

	result, err := reopened.Save(ctx, PlayerState{Gold: 5})
	if err == nil {
		t.Fatal("Save on a read-only database succeeded, want an error")
	}
	if result.OK {
		t.Error("SaveResult.OK = true, want false on a failed save")
	}
}
