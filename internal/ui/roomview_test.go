package ui

import (
	"errors"
	"nookbuddy/internal/game"
	"nookbuddy/internal/input"
	"nookbuddy/internal/storage"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func sized(t *testing.T, m Model, width, height int) Model {
	t.Helper()
	m, _ = send(t, m, tea.WindowSizeMsg{Width: width, Height: height})
	return m
}

func savingModel(source InputSource, state storage.PlayerState) Model {
	m, _ := loopModel(&countingSaver{}, source, state)
	return m
}

func TestModel_ViewShowsTooSmallBelowMinimum(t *testing.T) {
	for _, size := range [][2]int{{99, 40}, {120, 27}, {99, 27}, {80, 20}} {
		out := plain(sized(t, newTestModel(), size[0], size[1]).View())
		if !strings.Contains(out, "Terminal too small (need 100×28)") {
			t.Errorf("%dx%d: too-small message missing", size[0], size[1])
		}
		if strings.Contains(out, "Projects ready") || strings.Contains(out, "Sell body") || strings.Contains(out, "q quit") {
			t.Errorf("%dx%d: panel content rendered behind the too-small message", size[0], size[1])
		}
	}
}

func TestModel_TooSmallMessageIsCentered(t *testing.T) {
	lines := strings.Split(plain(sized(t, newTestModel(), 80, 21).View()), "\n")
	if len(lines) != 21 {
		t.Fatalf("too-small view has %d rows, want 21", len(lines))
	}
	row := lines[10]
	if !strings.Contains(row, "Terminal too small") {
		t.Fatalf("message is not on the middle row: %q", row)
	}
	left := len(row) - len(strings.TrimLeft(row, " "))
	if left < 15 {
		t.Errorf("message starts at column %d, want it centered", left)
	}
}

func TestModel_ViewRendersNormallyAtMinimumAndAbove(t *testing.T) {
	for _, size := range [][2]int{{100, 28}, {140, 40}} {
		out := plain(sized(t, newTestModel(), size[0], size[1]).View())
		if strings.Contains(out, "Terminal too small") {
			t.Errorf("%dx%d: too-small message shown", size[0], size[1])
		}
		for _, want := range []string{"Projects ready", "Sell body", "q quit"} {
			if !strings.Contains(out, want) {
				t.Errorf("%dx%d: %q missing", size[0], size[1], want)
			}
		}
		if lines := strings.Split(out, "\n"); len(lines) != size[1] {
			t.Errorf("%dx%d: view has %d rows", size[0], size[1], len(lines))
		}
	}
}

func TestModel_ViewRendersNormallyWhenSizeUnknown(t *testing.T) {
	out := plain(newTestModel().View())
	if strings.Contains(out, "Terminal too small") || !strings.Contains(out, "Projects ready") {
		t.Error("an unknown size must render the normal UI")
	}
}

func TestModel_ResizeBackRestoresUI(t *testing.T) {
	m := sized(t, newTestModel(), 80, 20)
	if !strings.Contains(plain(m.View()), "Terminal too small") {
		t.Fatal("too-small message missing at 80x20")
	}
	m = sized(t, m, 120, 40)
	if out := plain(m.View()); strings.Contains(out, "Terminal too small") || !strings.Contains(out, "Projects ready") {
		t.Error("the UI did not come back after resizing to 120x40")
	}
}

func TestModel_LeftPanelIsVisibleOnEveryScreen(t *testing.T) {
	m := sized(t, savingModel(newFakeSource(input.Available), storage.PlayerState{Gold: 42}), 120, 30)
	for _, screenKey := range []string{"1", "2", "3"} {
		m = press(t, m, screenKey)
		out := plain(m.View())
		for _, want := range []string{"Clicks", "Keys", "Projects ready", "Gold: 42", "┌──"} {
			if !strings.Contains(out, want) {
				t.Errorf("screen %s: room panel missing %q", screenKey, want)
			}
		}
	}
}

func TestModel_InitSchedulesAnimationTickEvenWithoutSource(t *testing.T) {
	m, sched := loopModel(nil, nil, storage.PlayerState{})
	if !contains(collect(m.Init()), animationTickMsg{}) {
		t.Fatal("Init did not schedule the animation tick")
	}
	if after, _ := sched.last(animationTickMsg{}); after != 250*time.Millisecond {
		t.Errorf("animation interval = %v, want 250ms", after)
	}
}

func TestModel_AnimationTickReschedulesItself(t *testing.T) {
	m, _ := loopModel(nil, nil, storage.PlayerState{Gold: 9, ClicksProgress: 3})
	next, cmd := send(t, m, animationTickMsg{})
	if !contains(collect(cmd), animationTickMsg{}) {
		t.Error("animation tick did not reschedule itself")
	}
	if next.state.Gold != 9 || next.state.ClicksProgress != 3 || next.dirty {
		t.Error("animation tick changed the game state")
	}
}

func TestModel_WithNoticeShowsRecoveryUntilNavigationKey(t *testing.T) {
	m := savingModel(newFakeSource(input.Available), storage.PlayerState{}).WithNotice(storage.NoticeRecoveredFromCorruption)
	shows := func(m Model) bool { return strings.Contains(plain(m.View()), "Previous save was unreadable") }

	if !shows(m) {
		t.Fatal("recovery notice missing on the first frame")
	}
	for _, k := range []string{"enter", "4", "x"} {
		if m = press(t, m, k); !shows(m) {
			t.Fatalf("%q dismissed the recovery notice", k)
		}
	}
	if m = press(t, m, "down"); shows(m) {
		t.Error("a navigation key did not dismiss the recovery notice")
	}
}

func TestModel_EveryNavigationKeyDismissesRecovery(t *testing.T) {
	for _, k := range []string{"1", "2", "3", "tab", "shift+tab", "up", "down", "j", "k"} {
		m := savingModel(newFakeSource(input.Available), storage.PlayerState{}).WithNotice(storage.NoticeRecoveredFromCorruption)
		if m = press(t, m, k); m.recovered {
			t.Errorf("%q did not dismiss the recovery notice", k)
		}
	}
}

func TestModel_WithNoticeIgnoresNoticeNone(t *testing.T) {
	if savingModel(nil, storage.PlayerState{}).WithNotice(storage.NoticeNone).recovered {
		t.Error("NoticeNone set the recovery flag")
	}
}

func TestModel_MemoryOnlyNoticeIsPersistent(t *testing.T) {
	m, _ := loopModel(nil, newFakeSource(input.Available), storage.PlayerState{})
	m = press(t, m, "down", "2", "tab", "up", "j")
	if !strings.Contains(plain(m.View()), "Cannot write save file") {
		t.Error("memory-only notice disappeared after navigation")
	}
}

func TestModel_HookNoticeFollowsStatus(t *testing.T) {
	source := newFakeSource(input.Unavailable)
	m := savingModel(source, storage.PlayerState{})
	if !strings.Contains(plain(m.View()), "Input capture unavailable") {
		t.Fatal("hook notice missing on the first frame")
	}

	source.setStatus(input.Available)
	m, _ = send(t, m, inputTickMsg{})
	if strings.Contains(plain(m.View()), "Input capture unavailable") {
		t.Error("hook notice shown while the hook is available")
	}
}

func TestModel_RecoveryNoticeAndSaveFailureFooterCoexist(t *testing.T) {
	captureLogs(t)
	saver := &countingSaver{err: errors.New("disk full")}
	m, _ := loopModel(saver, newFakeSource(input.Available), storage.PlayerState{Gold: 60})
	m = m.WithNotice(storage.NoticeRecoveredFromCorruption)
	m.screens[0].(*fakeScreen).goldDelta = -10

	m, cmd := send(t, m, key("enter"))
	m, _ = send(t, m, saveResults(collect(cmd))[0])

	out := plain(m.View())
	if !strings.Contains(out, "Previous save was unreadable") || !strings.Contains(out, "Last save failed — retrying") {
		t.Errorf("recovery notice and save-failure footer must render together:\n%s", out)
	}
}

func TestModel_LoadedGoldAndCosmeticsRenderOnTheFirstFrame(t *testing.T) {
	state := storage.PlayerState{Gold: 1234, OwnedCosmetics: []string{"painting", "rug"}}
	out := plain(savingModel(newFakeSource(input.Available), state).View())
	for _, want := range []string{"Gold: 1234", cosmeticArt["painting"], cosmeticArt["rug"]} {
		if !strings.Contains(out, want) {
			t.Errorf("first frame missing %q", want)
		}
	}
	if strings.Contains(out, cosmeticArt["window"]) {
		t.Error("an unowned cosmetic is drawn")
	}
}

func TestModel_LastInputDrivesWorkingAndSleeping(t *testing.T) {
	clock := frameBase
	m := savingModel(newFakeSource(input.Available), storage.PlayerState{})
	m.now = func() time.Time { return clock }
	asleep := func(m Model) bool { return strings.Contains(plain(m.View()), "(-_-)") }

	if !asleep(m) {
		t.Fatal("character is awake before any input")
	}
	m, _ = send(t, m, input.InputEventMsg{Events: []input.Event{{Type: input.Key}}})
	if asleep(m) {
		t.Fatal("character still asleep right after input")
	}

	clock = clock.Add(9 * time.Second)
	if asleep(m) {
		t.Fatal("character fell asleep before 10s")
	}
	clock = clock.Add(time.Second)
	if !asleep(m) {
		t.Fatal("character still awake 10s after the last input")
	}

	m, _ = send(t, m, input.InputEventMsg{Events: []input.Event{{Type: input.Click}}})
	if asleep(m) {
		t.Error("the next input did not wake the character on the next render")
	}
}

func TestModel_BarsAndProjectsFollowAppliedEvents(t *testing.T) {
	state := storage.PlayerState{}
	for range 100 {
		state = game.ApplyEvent(state, input.Event{Type: input.Click})
	}
	for range 450 {
		state = game.ApplyEvent(state, input.Event{Type: input.Key})
	}

	out := plain(savingModel(newFakeSource(input.Available), state).View())
	for _, want := range []string{"Clicks  ░░░░░░░░░░  0/100", "Keys    █████░░░░░  150/300", "Projects ready: 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestModel_ProjectCompletionHighlightsForOneSecond(t *testing.T) {
	forceANSI(t)
	clock := frameBase
	m := savingModel(newFakeSource(input.Available), storage.PlayerState{ClicksProgress: 99, KeysProgress: 300})
	m.now = func() time.Time { return clock }

	m, _ = send(t, m, input.InputEventMsg{Events: []input.Event{{Type: input.Click}}})
	if !strings.Contains(m.View(), Accent.Bold(true).Render("Projects ready: 1")) {
		t.Fatal("projects line is not highlighted right after the completion")
	}
	clock = clock.Add(time.Second)
	if strings.Contains(m.View(), Accent.Bold(true).Render("Projects ready: 1")) {
		t.Error("projects line still highlighted after 1 second")
	}
}
