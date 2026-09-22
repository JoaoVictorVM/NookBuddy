package config

import "time"

const (
	ClicksBarLimit = 100
	KeysBarLimit   = 300
)

const (
	BaseClickContribution = 1
	BaseKeyContribution   = 1
)

const BaseSaleValue = 10

const (
	UpgradeBaseCost       = 50
	UpgradeCostMultiplier = 1.5
)

const (
	UpgradeContributionIncrement = 1
	UpgradeValueIncrement        = 5
)

const CosmeticPrice = 100

const (
	IdleThreshold    = 10 * time.Second
	AutosaveInterval = 30 * time.Second
)
