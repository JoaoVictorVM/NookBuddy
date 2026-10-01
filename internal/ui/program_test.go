package ui

import (
	"context"
	"errors"
	"io"
	"nookbuddy/internal/storage"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func runProgram(t *testing.T, m Model, input string) (Model, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	program := tea.NewProgram(m,
		tea.WithContext(ctx),
		tea.WithInput(strings.NewReader(input)),
		tea.WithOutput(io.Discard),
		tea.WithoutSignalHandler(),
	)
	final, err := program.Run()
	if errors.Is(err, tea.ErrProgramKilled) && ctx.Err() != nil {
		t.Fatalf("program did not finish within 5s: %v", err)
	}
	if final == nil {
		return m, err
	}
	return final.(Model), err
}

func TestProgram_QuitSavesAndExitsCleanly(t *testing.T) {
	store, _ := openTempStore(t)
	m := NewModel(store, storage.PlayerState{Gold: 42}, threeScreens())

	final, err := runProgram(t, m, "2q")
	if err != nil {
		t.Fatalf("Run returned %v, want a clean exit", err)
	}
	if !final.quitting || final.activeScreen().Title() != "Upgrades" {
		t.Errorf("final model: quitting=%v active=%q, want true/Upgrades", final.quitting, final.activeScreen().Title())
	}

	got, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Gold != 42 {
		t.Errorf("Gold = %d, want 42 saved on quit", got.Gold)
	}
}

func assertCrashRecorded(t *testing.T, m Model, err error, value, frame string) {
	t.Helper()
	if !errors.Is(err, tea.ErrProgramPanic) {
		t.Fatalf("Run error = %v, want tea.ErrProgramPanic", err)
	}
	recovered, stack, ok := m.Crash()
	if !ok {
		t.Fatal("panic was not recorded for the crash log")
	}
	if recovered != value {
		t.Errorf("recorded value = %v, want %q", recovered, value)
	}
	if !strings.Contains(string(stack), frame) {
		t.Errorf("recorded stack does not reach the panic origin %q:\n%s", frame, stack)
	}
}

func TestProgram_PanicInUpdateIsRecordedForCrashLog(t *testing.T) {
	screens := threeScreens()
	screens[0].(*fakeScreen).panicOnEnter = true
	m := NewModel(nil, storage.PlayerState{}, screens)

	_, err := runProgram(t, m, "\r")
	assertCrashRecorded(t, m, err, "activate exploded", "(*fakeScreen).Activate")
}

func TestProgram_PanicInCommandIsRecordedForCrashLog(t *testing.T) {
	screens := threeScreens()
	screens[0].(*fakeScreen).activateCmd = func() tea.Msg { panic("command exploded") }
	m := NewModel(nil, storage.PlayerState{}, screens)

	_, err := runProgram(t, m, "\r")
	assertCrashRecorded(t, m, err, "command exploded", "TestProgram_PanicInCommandIsRecordedForCrashLog")
}

func TestProgram_PanicInViewIsRecordedForCrashLog(t *testing.T) {
	screens := threeScreens()
	screens[0].(*fakeScreen).panicOnView = true
	m := NewModel(nil, storage.PlayerState{}, screens)

	_, err := runProgram(t, m, "")
	assertCrashRecorded(t, m, err, "view exploded", "(*fakeScreen).View")
}

func TestProgram_OnlyTheFirstPanicIsRecorded(t *testing.T) {
	recorder := &crashRecorder{}
	recorder.record("first")
	recorder.record("second")
	if value, _, _ := recorder.report(); value != "first" {
		t.Errorf("recorded = %v, want the first panic", value)
	}
}
