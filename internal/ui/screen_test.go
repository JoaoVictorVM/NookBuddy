package ui

import "testing"

func TestClamp_WithinBounds(t *testing.T) {
	for _, selected := range []int{0, 2, 4} {
		if got := clamp(selected, 5); got != selected {
			t.Errorf("clamp(%d, 5) = %d, want %d", selected, got, selected)
		}
	}
}

func TestClamp_AboveUpperBound(t *testing.T) {
	if got := clamp(9, 5); got != 4 {
		t.Errorf("clamp(9, 5) = %d, want 4", got)
	}
}

func TestClamp_BelowLowerBound(t *testing.T) {
	if got := clamp(-1, 5); got != 0 {
		t.Errorf("clamp(-1, 5) = %d, want 0", got)
	}
}

func TestClamp_EmptyList(t *testing.T) {
	for _, selected := range []int{-1, 0, 3} {
		if got := clamp(selected, 0); got != 0 {
			t.Errorf("clamp(%d, 0) = %d, want 0", selected, got)
		}
	}
}
