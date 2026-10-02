package game

import "testing"

func TestCosmeticPrice_MatchesConfig(t *testing.T) {
	if got := CosmeticPrice(); got != 100 {
		t.Errorf("CosmeticPrice() = %d, want 100", got)
	}
}
