package game

import (
	"nookbuddy/internal/config"
	"nookbuddy/internal/input"
	"nookbuddy/internal/storage"
	"testing"
	"time"
)

var (
	click = input.Event{Type: input.Click, Timestamp: time.Now()}
	key   = input.Event{Type: input.Key, Timestamp: time.Now()}
)

func applyN(state storage.PlayerState, event input.Event, n int) storage.PlayerState {
	for range n {
		state = ApplyEvent(state, event)
	}
	return state
}

func TestApplyEvent_ClickIncrementsClicksProgressByContribution(t *testing.T) {
	got := ApplyEvent(storage.PlayerState{ClicksProgress: 10, KeysProgress: 7}, click)
	if got.ClicksProgress != 10+ClickContribution(0) || got.ClicksProgress != 11 {
		t.Errorf("ClicksProgress = %d, want 11", got.ClicksProgress)
	}
	if got.KeysProgress != 7 {
		t.Errorf("KeysProgress = %d, want unchanged 7", got.KeysProgress)
	}
}

func TestApplyEvent_KeyIncrementsKeysProgressByContribution(t *testing.T) {
	got := ApplyEvent(storage.PlayerState{KeysProgress: 20, UpgradeKeysLevel: 2, ClicksProgress: 4}, key)
	if got.KeysProgress != 20+KeyContribution(2) || got.KeysProgress != 23 {
		t.Errorf("KeysProgress = %d, want 23", got.KeysProgress)
	}
	if got.ClicksProgress != 4 {
		t.Errorf("ClicksProgress = %d, want unchanged 4", got.ClicksProgress)
	}
}

func TestApplyEvent_UsesTheCurrentUpgradeLevel(t *testing.T) {
	state := storage.PlayerState{}
	state = ApplyEvent(state, click)
	state.UpgradeClicksLevel = 1
	state = ApplyEvent(state, click)
	if state.ClicksProgress != 3 {
		t.Errorf("ClicksProgress = %d, want 1 (Lv 0) + 2 (Lv 1) = 3", state.ClicksProgress)
	}
}

func TestApplyEvent_ClicksProgressCapsAtLimit(t *testing.T) {
	got := ApplyEvent(storage.PlayerState{ClicksProgress: 99, UpgradeClicksLevel: 4}, click)
	if got.ClicksProgress != config.ClicksBarLimit {
		t.Errorf("ClicksProgress = %d, want exactly %d", got.ClicksProgress, config.ClicksBarLimit)
	}
}

func TestApplyEvent_KeysProgressCapsAtLimit(t *testing.T) {
	got := ApplyEvent(storage.PlayerState{KeysProgress: 298, UpgradeKeysLevel: 9}, key)
	if got.KeysProgress != config.KeysBarLimit {
		t.Errorf("KeysProgress = %d, want exactly %d", got.KeysProgress, config.KeysBarLimit)
	}
}

func TestApplyEvent_LevelZeroFillsBarsAtTheirLimits(t *testing.T) {
	clicks := applyN(storage.PlayerState{}, click, 100)
	if clicks.ClicksProgress != 100 {
		t.Errorf("100 clicks → %d, want 100", clicks.ClicksProgress)
	}
	keys := applyN(storage.PlayerState{}, key, 300)
	if keys.KeysProgress != 300 {
		t.Errorf("300 keys → %d, want 300", keys.KeysProgress)
	}
}

func TestApplyEvent_CompletesProjectAndResetsBothBars(t *testing.T) {
	got := ApplyEvent(storage.PlayerState{ClicksProgress: 99, KeysProgress: 300, ProjectsReady: 4}, click)
	if got.ProjectsReady != 5 {
		t.Errorf("ProjectsReady = %d, want 5", got.ProjectsReady)
	}
	if got.ClicksProgress != 0 || got.KeysProgress != 0 {
		t.Errorf("bars = %d/%d, want both reset to 0", got.ClicksProgress, got.KeysProgress)
	}
}

func TestApplyEvent_ProjectNeedsBothBars(t *testing.T) {
	state := applyN(storage.PlayerState{}, click, 100)
	if state.ProjectsReady != 0 {
		t.Fatalf("100 clicks and 0 keys → %d projects, want 0", state.ProjectsReady)
	}
	state = applyN(state, key, 300)
	if state.ProjectsReady != 1 || state.ClicksProgress != 0 || state.KeysProgress != 0 {
		t.Errorf("after 300 keys: projects=%d bars=%d/%d, want 1 and both 0", state.ProjectsReady, state.ClicksProgress, state.KeysProgress)
	}
}

func TestApplyEvent_NoCompletionWhenOnlyOneBarIsFull(t *testing.T) {
	got := ApplyEvent(storage.PlayerState{ClicksProgress: 100}, click)
	if got.ProjectsReady != 0 {
		t.Errorf("ProjectsReady = %d, want 0", got.ProjectsReady)
	}
	if got.ClicksProgress != 100 {
		t.Errorf("ClicksProgress = %d, want 100 kept", got.ClicksProgress)
	}
}

func TestApplyEvent_ExtraClicksAreNotCarriedOverAfterReset(t *testing.T) {
	state := applyN(storage.PlayerState{}, click, 250)
	state = applyN(state, key, 300)
	if state.ProjectsReady != 1 || state.ClicksProgress != 0 {
		t.Errorf("after 250 clicks then 300 keys: projects=%d clicks=%d, want 1 and 0 (no carry-over)", state.ProjectsReady, state.ClicksProgress)
	}
}

func TestApplyEvent_ProjectsReadyHasNoUpperLimitAcrossRepeatedCompletions(t *testing.T) {
	state := storage.PlayerState{}
	for range 3 {
		state = applyN(state, click, 100)
		state = applyN(state, key, 300)
	}
	if state.ProjectsReady != 3 {
		t.Errorf("ProjectsReady = %d, want 3", state.ProjectsReady)
	}
}

func TestApplyEvent_LeavesUnrelatedFieldsAlone(t *testing.T) {
	state := storage.PlayerState{Gold: 42, UpgradeValueLevel: 3, OwnedCosmetics: []string{"rug"}}
	got := ApplyEvent(ApplyEvent(state, click), key)
	if got.Gold != 42 || got.UpgradeValueLevel != 3 || len(got.OwnedCosmetics) != 1 {
		t.Errorf("unrelated fields changed: %+v", got)
	}
}

func TestApplyEvent_UnknownEventTypeChangesNothing(t *testing.T) {
	state := storage.PlayerState{ClicksProgress: 5, KeysProgress: 6}
	got := ApplyEvent(state, input.Event{Type: input.EventType(99)})
	if got.ClicksProgress != 5 || got.KeysProgress != 6 {
		t.Errorf("unknown event changed bars to %d/%d", got.ClicksProgress, got.KeysProgress)
	}
}

func TestIsProjectComplete_TrueOnlyWhenBothBarsAreAtOrAboveLimit(t *testing.T) {
	cases := []struct {
		clicks, keys int
		want         bool
	}{
		{100, 300, true},
		{101, 301, true},
		{100, 299, false},
		{99, 300, false},
		{0, 0, false},
		{100, 0, false},
		{0, 300, false},
	}
	for _, tc := range cases {
		if got := IsProjectComplete(storage.PlayerState{ClicksProgress: tc.clicks, KeysProgress: tc.keys}); got != tc.want {
			t.Errorf("IsProjectComplete(%d, %d) = %v, want %v", tc.clicks, tc.keys, got, tc.want)
		}
	}
}
