package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var frameBase = time.UnixMilli(2400 * 1_000_000_000)

var cosmeticArt = map[string]string{
	"window":    "╔══╦══╗",
	"painting":  "│ ~^~ │",
	"bookshelf": "│▌▐▌│",
	"flower":    `\|/`,
	"rug":       "░▒▓▓▓▓▓▓▓▓▓▓▓▓▒░",
}

func TestSleeping_FalseWithinThreshold(t *testing.T) {
	if sleeping(frameBase, frameBase.Add(-9*time.Second)) {
		t.Error("asleep 9s after the last input, want awake")
	}
}

func TestSleeping_TrueAtAndBeyondThreshold(t *testing.T) {
	for _, idle := range []time.Duration{10 * time.Second, time.Minute} {
		if !sleeping(frameBase, frameBase.Add(-idle)) {
			t.Errorf("awake %v after the last input, want asleep", idle)
		}
	}
}

func TestSleeping_TrueWhenNeverActive(t *testing.T) {
	if !sleeping(frameBase, time.Time{}) {
		t.Error("awake with no input since launch, want asleep")
	}
}

func TestWorkingFrames_CycleEvery500ms(t *testing.T) {
	var frames []string
	for i := range 4 {
		at := frameBase.Add(time.Duration(i) * 500 * time.Millisecond)
		frames = append(frames, plain(renderScene(at, at, nil)))
	}
	if frames[0] == frames[1] || frames[1] == frames[2] || frames[0] == frames[2] {
		t.Error("the three working frames are not distinct")
	}
	if frames[3] != frames[0] {
		t.Error("the working animation does not wrap back to the first frame")
	}
	if a, b := plain(renderScene(frameBase, frameBase, nil)), plain(renderScene(frameBase.Add(499*time.Millisecond), frameBase, nil)); a != b {
		t.Error("the working frame changed before 500ms")
	}
}

func TestSleepingFrames_CycleEvery800ms(t *testing.T) {
	want := []string{"z", "Z", "Z Z", "z"}
	for i, snore := range want {
		at := frameBase.Add(time.Duration(i) * 800 * time.Millisecond)
		if got := snoreFrames[snoreFrame(at)].lines[0]; got != snore {
			t.Errorf("snore frame %d = %q, want %q", i, got, snore)
		}
		if scene := plain(renderScene(at, time.Time{}, nil)); !strings.Contains(scene, snore) || !strings.Contains(scene, "(-_-)") {
			t.Errorf("sleeping scene %d does not show %q above the sleeping pose", i, snore)
		}
	}
}

func TestScene_SleepingAndWorkingPosesDiffer(t *testing.T) {
	working := plain(renderScene(frameBase, frameBase, nil))
	asleep := plain(renderScene(frameBase, frameBase.Add(-10*time.Second), nil))
	if strings.Contains(working, "(-_-)") || !strings.Contains(asleep, "(-_-)") {
		t.Error("the sleeping pose must appear only after the idle threshold")
	}
	if strings.ContainsAny(working, "zZ") {
		t.Error("a working character shows snoring Zs")
	}
}

func TestRoom_UnownedSlotsRenderAsBlank(t *testing.T) {
	scene := plain(renderScene(frameBase, frameBase, nil))
	for id, art := range cosmeticArt {
		if strings.Contains(scene, art) {
			t.Errorf("unowned %s is drawn", id)
		}
	}
}

func TestRoom_EachOwnedCosmeticAppearsInItsSlot(t *testing.T) {
	for id := range cosmeticArt {
		scene := plain(renderScene(frameBase, frameBase, []string{id}))
		for other, art := range cosmeticArt {
			if present := strings.Contains(scene, art); present != (other == id) {
				t.Errorf("owning only %s: %s drawn = %v", id, other, present)
			}
		}
	}
}

func TestRoom_AllFiveOwnedFitTheRoom(t *testing.T) {
	scene := renderScene(frameBase, frameBase.Add(-time.Minute), []string{"window", "bookshelf", "flower", "painting", "rug"})
	for id, art := range cosmeticArt {
		if !strings.Contains(plain(scene), art) {
			t.Errorf("%s missing when all five are owned", id)
		}
	}
	lines := strings.Split(scene, "\n")
	if len(lines) != roomHeight {
		t.Errorf("room has %d rows, want %d", len(lines), roomHeight)
	}
	for i, line := range lines {
		if w := lipgloss.Width(line); w != roomWidth {
			t.Errorf("room row %d is %d columns, want exactly %d", i, w, roomWidth)
		}
	}
}

func TestRoom_UnknownCosmeticIsIgnored(t *testing.T) {
	if plain(renderScene(frameBase, frameBase, []string{"spaceship"})) != plain(renderScene(frameBase, frameBase, nil)) {
		t.Error("an unknown cosmetic ID changed the room")
	}
}

func TestRoom_PurchaseShowsOnNextRender(t *testing.T) {
	owned := []string{}
	before := plain(renderScene(frameBase, frameBase, owned))
	owned = append(owned, "rug")
	after := plain(renderScene(frameBase, frameBase, owned))
	if strings.Contains(before, cosmeticArt["rug"]) || !strings.Contains(after, cosmeticArt["rug"]) {
		t.Error("the rug did not appear on the render right after it was added")
	}
}
