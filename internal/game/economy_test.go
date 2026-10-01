package game

import (
	"math"
	"math/big"
	"testing"
)

func exactCost(level int) *big.Int {
	numerator := new(big.Int).Mul(big.NewInt(50), new(big.Int).Exp(big.NewInt(3), big.NewInt(int64(level)), nil))
	denominator := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(level)), nil)
	return numerator.Quo(numerator, denominator)
}

func TestUpgradeCost_FollowsFormulaForFirstLevels(t *testing.T) {
	want := []int64{50, 75, 112, 168, 253, 379}
	for level, cost := range want {
		if got := UpgradeCost(level); got != cost {
			t.Errorf("UpgradeCost(%d) = %d, want %d", level, got, cost)
		}
	}
}

func TestUpgradeCost_IsExactFloorUpToTheCap(t *testing.T) {
	limit := big.NewInt(math.MaxInt64)
	for level := 0; ; level++ {
		exact := exactCost(level)
		if exact.Cmp(limit) >= 0 {
			if got := UpgradeCost(level); got != math.MaxInt64 {
				t.Errorf("UpgradeCost(%d) = %d, want the cap once the exact cost reaches int64", level, got)
			}
			return
		}
		if got := UpgradeCost(level); got != exact.Int64() {
			t.Fatalf("UpgradeCost(%d) = %d, want exact floor %s", level, got, exact)
		}
	}
}

func TestUpgradeCost_CapsAtInt64Max(t *testing.T) {
	for _, level := range []int{120, 400, 10_000} {
		if got := UpgradeCost(level); got != math.MaxInt64 {
			t.Errorf("UpgradeCost(%d) = %d, want math.MaxInt64", level, got)
		}
	}
}

func TestUpgradeCost_NeverDecreases(t *testing.T) {
	previous := UpgradeCost(0)
	for level := 1; level < 200; level++ {
		current := UpgradeCost(level)
		if current < previous {
			t.Fatalf("UpgradeCost(%d) = %d is below UpgradeCost(%d) = %d", level, current, level-1, previous)
		}
		previous = current
	}
}

func TestClickContribution_AddsOnePerLevel(t *testing.T) {
	for level, want := range map[int]int{0: 1, 1: 2, 3: 4} {
		if got := ClickContribution(level); got != want {
			t.Errorf("ClickContribution(%d) = %d, want %d", level, got, want)
		}
	}
}

func TestKeyContribution_AddsOnePerLevel(t *testing.T) {
	for level, want := range map[int]int{0: 1, 1: 2, 3: 4} {
		if got := KeyContribution(level); got != want {
			t.Errorf("KeyContribution(%d) = %d, want %d", level, got, want)
		}
	}
}

func TestSaleValue_AddsFiveGoldPerLevel(t *testing.T) {
	for level, want := range map[int]int{0: 10, 1: 15, 2: 20} {
		if got := SaleValue(level); got != want {
			t.Errorf("SaleValue(%d) = %d, want %d", level, got, want)
		}
	}
}
