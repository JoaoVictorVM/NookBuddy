package main

import (
	"fmt"
	"log/slog"
	"nookbuddy/internal/logging"
	"nookbuddy/internal/version"
	"os"
	"runtime/debug"
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
			code = crash(recovered)
		}
	}()

	slog.Info("nookbuddy: starting", "version", version.Version, "commit", version.Commit, "date", version.Date)
	return run()
}

func crash(recovered any) int {
	stack := debug.Stack()
	slog.Error("nookbuddy: panic", "panic", recovered, "stack", string(stack))
	dir, _ := logging.Dir()
	logging.ReportCrash(dir, recovered, stack, version.String(), os.Stderr)
	return 1
}

func run() int {
	fmt.Println("nookbuddy: scaffold ready")
	return 0
}
