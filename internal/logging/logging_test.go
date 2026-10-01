package logging

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func isolate(t *testing.T) {
	t.Helper()
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })
}

func setupIn(t *testing.T, dir string, fallback io.Writer) {
	t.Helper()
	isolate(t)
	closer := setup(dir, fallback)
	t.Cleanup(func() { _ = closer.Close() })
}

func readLog(t *testing.T, dir string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(dir, logFileName))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	return string(content)
}

func TestDir_EndsInAppFolder(t *testing.T) {
	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if filepath.Base(dir) != "NookBuddy" {
		t.Errorf("Dir = %q, want it to end in NookBuddy", dir)
	}
}

func TestSetup_CreatesLogDirectoryAndFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "NookBuddy")
	setupIn(t, dir, io.Discard)

	slog.Info("hello", "clicks", 3)

	lines := strings.Split(strings.TrimSpace(readLog(t, dir)), "\n")
	if len(lines) != 1 {
		t.Fatalf("log has %d lines, want 1", len(lines))
	}
	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("log line is not JSON: %v (%q)", err, lines[0])
	}
	if entry["msg"] != "hello" || entry["level"] != "INFO" || entry["clicks"] != float64(3) {
		t.Errorf("entry = %v, want msg=hello level=INFO clicks=3", entry)
	}
}

func TestSetup_RotatesOversizedLogOnStartup(t *testing.T) {
	dir := t.TempDir()
	old := bytes.Repeat([]byte("x"), maxLogSize+1)
	if err := os.WriteFile(filepath.Join(dir, logFileName), old, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, logFileName+".1"), []byte("older"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	setupIn(t, dir, io.Discard)
	slog.Info("fresh start")

	rotated, err := os.ReadFile(filepath.Join(dir, logFileName+".1"))
	if err != nil {
		t.Fatalf("ReadFile rotated: %v", err)
	}
	if !bytes.Equal(rotated, old) {
		t.Errorf("app.log.1 has %d bytes, want the previous %d-byte app.log", len(rotated), len(old))
	}
	current := readLog(t, dir)
	if strings.Contains(current, "xxx") || !strings.Contains(current, "fresh start") {
		t.Errorf("app.log = %q, want only the new entry", current)
	}
}

func TestSetup_DoesNotRotateUndersizedLog(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, logFileName), []byte("{\"msg\":\"previous\"}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	setupIn(t, dir, io.Discard)
	slog.Info("appended")

	content := readLog(t, dir)
	if !strings.HasPrefix(content, "{\"msg\":\"previous\"}\n") || !strings.Contains(content, "appended") {
		t.Errorf("app.log = %q, want previous content kept and the new entry appended", content)
	}
	if _, err := os.Stat(filepath.Join(dir, logFileName+".1")); err == nil {
		t.Error("app.log.1 exists, want no rotation for a small log")
	}
}

func TestSetup_RotatesOnlyAboveFiveMegabytes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, logFileName), bytes.Repeat([]byte("x"), maxLogSize), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	setupIn(t, dir, io.Discard)

	if _, err := os.Stat(filepath.Join(dir, logFileName+".1")); err == nil {
		t.Error("a log of exactly 5MB was rotated, want rotation only when it exceeds 5MB")
	}
	if maxLogSize != 5*1024*1024 {
		t.Errorf("maxLogSize = %d, want 5MB", maxLogSize)
	}
}

func TestSetup_DebugLevelViaEnvVar(t *testing.T) {
	t.Run("debug enabled", func(t *testing.T) {
		t.Setenv("NOOKBUDDY_LOG", "debug")
		dir := t.TempDir()
		setupIn(t, dir, io.Discard)
		slog.Debug("verbose detail")
		if !strings.Contains(readLog(t, dir), "verbose detail") {
			t.Error("debug entry missing with NOOKBUDDY_LOG=debug")
		}
	})

	t.Run("default is info", func(t *testing.T) {
		t.Setenv("NOOKBUDDY_LOG", "")
		dir := t.TempDir()
		setupIn(t, dir, io.Discard)
		slog.Debug("verbose detail")
		if strings.Contains(readLog(t, dir), "verbose detail") {
			t.Error("debug entry written without NOOKBUDDY_LOG=debug")
		}
	})

	t.Run("only exact lowercase debug", func(t *testing.T) {
		t.Setenv("NOOKBUDDY_LOG", "DEBUG")
		dir := t.TempDir()
		setupIn(t, dir, io.Discard)
		slog.Debug("verbose detail")
		if strings.Contains(readLog(t, dir), "verbose detail") {
			t.Error("NOOKBUDDY_LOG=DEBUG enabled debug, want only the exact value debug")
		}
	})
}

func TestSetup_FallsBackToStderrWhenDirNotWritable(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("file in the way"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	var stderr bytes.Buffer

	setupIn(t, filepath.Join(blocker, "NookBuddy"), &stderr)
	slog.Info("still running")

	out := stderr.String()
	if !strings.Contains(out, "not writable") {
		t.Errorf("stderr = %q, want the fallback warning", out)
	}
	if !strings.Contains(out, "still running") {
		t.Errorf("stderr = %q, want later entries to go to stderr", out)
	}
}
