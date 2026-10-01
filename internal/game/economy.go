package game

import (
	"math"
	"math/big"
	"nookbuddy/internal/config"
)

func UpgradeCost(level int) int64 {
	cost := new(big.Rat).SetInt64(config.UpgradeBaseCost)
	multiplier := new(big.Rat).SetFloat64(config.UpgradeCostMultiplier)
	limit := new(big.Rat).SetInt64(math.MaxInt64)

	for range level {
		cost.Mul(cost, multiplier)
		if cost.Cmp(limit) >= 0 {
			return math.MaxInt64
		}
	}
	return new(big.Int).Quo(cost.Num(), cost.Denom()).Int64()
}

func ClickContribution(level int) int {
	return config.BaseClickContribution + level*config.UpgradeContributionIncrement
}

func KeyContribution(level int) int {
	return config.BaseKeyContribution + level*config.UpgradeContributionIncrement
}

func SaleValue(level int) int {
	return config.BaseSaleValue + level*config.UpgradeValueIncrement
}
