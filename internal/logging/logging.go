package logging

import (
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

const (
	appDirName    = "NookBuddy"
	logFileName   = "app.log"
	maxLogSize    = 5 << 20
	levelEnvVar   = "NOOKBUDDY_LOG"
	debugLevelVal = "debug"
)

type nopCloser struct{}

func (nopCloser) Close() error { return nil }

func Dir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appDirName), nil
}

func Setup() io.Closer {
	dir, err := Dir()
	if err != nil {
		return fallBackTo(os.Stderr, err)
	}
	return setup(dir, os.Stderr)
}

func setup(dir string, fallback io.Writer) io.Closer {
	file, err := openLog(dir)
	if err != nil {
		return fallBackTo(fallback, err)
	}
	install(file)
	return file
}

func fallBackTo(w io.Writer, cause error) io.Closer {
	install(w)
	slog.Warn("nookbuddy: log directory not writable, logging to stderr", "error", cause)
	return nopCloser{}
}

func install(w io.Writer) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level()})))
}

func level() slog.Level {
	if os.Getenv(levelEnvVar) == debugLevelVal {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

func openLog(dir string) (*os.File, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, logFileName)
	if err := rotate(path); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
}

func rotate(path string) error {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Size() <= maxLogSize {
		return nil
	}
	return os.Rename(path, path+".1")
}
