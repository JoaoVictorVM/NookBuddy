package logging

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReportCrash_WritesCrashLogAndPointsToIt(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "NookBuddy")
	var stderr bytes.Buffer

	ReportCrash(dir, "boom", []byte("goroutine 1 [running]:\nmain.run()"), "NookBuddy v1.0.0 (abc1234, 2026-09-21)", &stderr)

	path := filepath.Join(dir, crashFileName)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("crash.log not written: %v", err)
	}
	for _, want := range []string{"time: ", "version: NookBuddy v1.0.0 (abc1234, 2026-09-21)", "panic: boom", "goroutine 1 [running]:", "main.run()"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("crash.log missing %q:\n%s", want, content)
		}
	}
	if got := stderr.String(); got != "NookBuddy crashed — details in "+path+"\n" {
		t.Errorf("stderr = %q, want the crash.log path", got)
	}
}

func TestReportCrash_AppendsToPreviousCrashes(t *testing.T) {
	dir := t.TempDir()
	var stderr bytes.Buffer

	ReportCrash(dir, "first", []byte("stack one"), "v", &stderr)
	ReportCrash(dir, "second", []byte("stack two"), "v", &stderr)

	content, err := os.ReadFile(filepath.Join(dir, crashFileName))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(content), "panic: first") || !strings.Contains(string(content), "panic: second") {
		t.Errorf("crash.log = %q, want both crashes kept", content)
	}
}

func TestReportCrash_PrintsStackToStderrWhenCrashLogFails(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("file in the way"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	var stderr bytes.Buffer

	ReportCrash(filepath.Join(blocker, "NookBuddy"), "boom", []byte("goroutine 1 [running]"), "v", &stderr)

	out := stderr.String()
	for _, want := range []string{"could not write crash.log", "panic: boom", "goroutine 1 [running]"} {
		if !strings.Contains(out, want) {
			t.Errorf("stderr missing %q:\n%s", want, out)
		}
	}
}

func TestReportCrash_EmptyDirFallsBackToStderr(t *testing.T) {
	var stderr bytes.Buffer
	ReportCrash("", "boom", []byte("stack"), "v", &stderr)
	if !strings.Contains(stderr.String(), "could not write crash.log") {
		t.Errorf("stderr = %q, want the fallback message", stderr.String())
	}
}

func TestWriteCrash_TimestampIsRFC3339(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 10, 1, 9, 30, 0, 0, time.UTC)

	path, err := writeCrash(dir, "boom", []byte("stack"), "v", at)
	if err != nil {
		t.Fatalf("writeCrash: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.HasPrefix(string(content), "time: 2026-10-01T09:30:00Z\n") {
		t.Errorf("crash.log starts with %q, want the RFC3339 timestamp", strings.SplitN(string(content), "\n", 2)[0])
	}
}
