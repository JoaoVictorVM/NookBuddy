package game

import (
	"math"
	"nookbuddy/internal/storage"
)

type SaleResult struct {
	Next   storage.PlayerState
	Sold   int
	Earned int
	Capped bool
}

func SaleTotal(projects, valueLevel int) int {
	total, _ := saleTotal(projects, valueLevel)
	return total
}

func saleTotal(projects, valueLevel int) (int, bool) {
	if projects <= 0 {
		return 0, false
	}
	value := SaleValue(valueLevel)
	if value > math.MaxInt/projects {
		return math.MaxInt, true
	}
	return projects * value, false
}

func Sell(state storage.PlayerState) SaleResult {
	total, capped := saleTotal(state.ProjectsReady, state.UpgradeValueLevel)
	earned := min(total, math.MaxInt-state.Gold)
	if earned < total {
		capped = true
	}

	next := state
	next.Gold += earned
	next.ProjectsReady = 0
	return SaleResult{Next: next, Sold: max(state.ProjectsReady, 0), Earned: earned, Capped: capped}
}
