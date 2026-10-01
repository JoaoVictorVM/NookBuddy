package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const crashFileName = "crash.log"

func ReportCrash(dir string, recovered any, stack []byte, build string, stderr io.Writer) {
	path, err := writeCrash(dir, recovered, stack, build, time.Now())
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "NookBuddy crashed — could not write crash.log (%v)\npanic: %v\n\n%s\n", err, recovered, stack)
		return
	}
	_, _ = fmt.Fprintf(stderr, "NookBuddy crashed — details in %s\n", path)
}

func writeCrash(dir string, recovered any, stack []byte, build string, now time.Time) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("no crash log directory")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, crashFileName)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return "", err
	}
	_, writeErr := fmt.Fprintf(file, "time: %s\nversion: %s\npanic: %v\n\n%s\n\n", now.Format(time.RFC3339), build, recovered, stack)
	if closeErr := file.Close(); writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		return "", writeErr
	}
	return path, nil
}
