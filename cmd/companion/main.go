package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"nookbuddy/internal/input"
	"nookbuddy/internal/logging"
	"nookbuddy/internal/storage"
	"nookbuddy/internal/ui"
	"nookbuddy/internal/ui/screens"
	"nookbuddy/internal/version"
	"os"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(version.String())
		return
	}
	os.Exit(start())
}

func start() (code int) {
	logs := logging.Setup()
	defer func() { _ = logs.Close() }()
	defer func() {
		if recovered := recover(); recovered != nil {
			code = crash(recovered, debug.Stack())
		}
	}()

	slog.Info("nookbuddy: starting", "version", version.Version, "commit", version.Commit, "date", version.Date)
	return run()
}

func crash(recovered any, stack []byte) int {
	slog.Error("nookbuddy: panic", "panic", recovered, "stack", string(stack))
	dir, _ := logging.Dir()
	logging.ReportCrash(dir, recovered, stack, version.String(), os.Stderr)
	return 1
}

func run() int {
	ctx := context.Background()

	store, state := openStore(ctx)
	var saver ui.Saver
	if store != nil {
		defer func() { _ = store.Close() }()
		saver = store
	}

	source := input.NewSource()
	_ = source.Start()
	defer source.Stop()

	model := ui.NewModel(saver, source, state, []ui.Screen{screens.Sell{}, screens.NewUpgrades(), screens.Customize{}})
	_, err := tea.NewProgram(model, tea.WithAltScreen()).Run()

	if errors.Is(err, tea.ErrProgramPanic) {
		if value, stack, ok := model.Crash(); ok {
			return crash(value, stack)
		}
	}
	if err != nil {
		slog.Error("nookbuddy: program stopped with an error", "error", err)
		return 1
	}
	return 0
}

func openStore(ctx context.Context) (*storage.Store, storage.PlayerState) {
	path, err := storage.DefaultPath()
	if err != nil {
		slog.Error("nookbuddy: cannot resolve save location, running without saving", "error", err)
		return nil, storage.PlayerState{}
	}

	store, notice, err := storage.Open(ctx, path)
	if err != nil {
		slog.Error("nookbuddy: cannot open save file, running without saving", "path", path, "error", err)
		return nil, storage.PlayerState{}
	}
	if notice == storage.NoticeRecoveredFromCorruption {
		slog.Warn("nookbuddy: previous save was unreadable and was moved aside", "path", path)
	}

	state, err := store.Load(ctx)
	if err != nil {
		slog.Error("nookbuddy: cannot load save file, running without saving", "path", path, "error", err)
		_ = store.Close()
		return nil, storage.PlayerState{}
	}
	return store, state
}
