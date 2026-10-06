package game

import (
	"math"
	"nookbuddy/internal/storage"
	"slices"
	"testing"
)

func TestSaleTotal_MultipliesProjectsByValue(t *testing.T) {
	if got := SaleTotal(3, 0); got != 30 {
		t.Errorf("SaleTotal(3, 0) = %d, want 30", got)
	}
	if got := SaleTotal(2, 2); got != 40 {
		t.Errorf("SaleTotal(2, 2) = %d, want 40", got)
	}
	if got := SaleTotal(0, 5); got != 0 {
		t.Errorf("SaleTotal(0, 5) = %d, want 0", got)
	}
}

func TestSaleTotal_CapsOnOverflow(t *testing.T) {
	for _, projects := range []int{math.MaxInt, math.MaxInt / 10} {
		if got := SaleTotal(projects, 1); got != math.MaxInt {
			t.Errorf("SaleTotal(%d, 1) = %d, want math.MaxInt", projects, got)
		}
	}
}

func TestSell_AddsEarnedGoldAndClearsProjects(t *testing.T) {
	sale := Sell(storage.PlayerState{Gold: 5, ProjectsReady: 3})
	if sale.Next.Gold != 35 || sale.Next.ProjectsReady != 0 {
		t.Errorf("after sale: gold=%d projects=%d, want 35/0", sale.Next.Gold, sale.Next.ProjectsReady)
	}
	if sale.Sold != 3 || sale.Earned != 30 || sale.Capped {
		t.Errorf("sale = %+v, want Sold 3, Earned 30, not capped", sale)
	}
}

func TestSell_UsesValueUpgradeLevel(t *testing.T) {
	sale := Sell(storage.PlayerState{ProjectsReady: 2, UpgradeValueLevel: 2})
	if sale.Earned != 40 || sale.Earned != 2*SaleValue(2) {
		t.Errorf("Earned = %d, want 40 (20 gold per project)", sale.Earned)
	}
}

func TestSell_SaturatesGoldAtMaxInt(t *testing.T) {
	sale := Sell(storage.PlayerState{Gold: math.MaxInt - 5, ProjectsReady: 3})
	if sale.Next.Gold != math.MaxInt || sale.Earned != 5 || !sale.Capped {
		t.Errorf("saturated sale = gold %d, earned %d, capped %v; want MaxInt, 5, true", sale.Next.Gold, sale.Earned, sale.Capped)
	}
	if sale.Next.ProjectsReady != 0 {
		t.Error("a capped sale must still clear the projects")
	}
}

func TestSell_CapsWhenTheTotalItselfOverflows(t *testing.T) {
	sale := Sell(storage.PlayerState{ProjectsReady: math.MaxInt})
	if sale.Next.Gold != math.MaxInt || !sale.Capped {
		t.Errorf("overflowing total: gold=%d capped=%v, want MaxInt/true", sale.Next.Gold, sale.Capped)
	}
}

func TestSell_LeavesOtherFieldsUntouched(t *testing.T) {
	state := storage.PlayerState{
		ClicksProgress: 40, KeysProgress: 90, ProjectsReady: 1, Gold: 7,
		UpgradeClicksLevel: 1, UpgradeKeysLevel: 2, UpgradeValueLevel: 3,
		OwnedCosmetics: []string{"rug", "window"},
	}
	next := Sell(state).Next
	if next.ClicksProgress != 40 || next.KeysProgress != 90 ||
		next.UpgradeClicksLevel != 1 || next.UpgradeKeysLevel != 2 || next.UpgradeValueLevel != 3 ||
		!slices.Equal(next.OwnedCosmetics, state.OwnedCosmetics) {
		t.Errorf("sale changed unrelated fields: %+v", next)
	}
}

func TestSell_WithZeroProjectsChangesNothing(t *testing.T) {
	sale := Sell(storage.PlayerState{Gold: 12})
	if sale.Next.Gold != 12 || sale.Sold != 0 || sale.Earned != 0 || sale.Capped {
		t.Errorf("empty sale = %+v, want no change", sale)
	}
}
