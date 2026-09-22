package config

import (
	"testing"
	"time"
)

func TestBarLimits(t *testing.T) {
	if ClicksBarLimit != 100 {
		t.Errorf("ClicksBarLimit = %d, want 100", ClicksBarLimit)
	}
	if KeysBarLimit != 300 {
		t.Errorf("KeysBarLimit = %d, want 300", KeysBarLimit)
	}
}

func TestBaseContributions(t *testing.T) {
	if BaseClickContribution != 1 {
		t.Errorf("BaseClickContribution = %d, want 1", BaseClickContribution)
	}
	if BaseKeyContribution != 1 {
		t.Errorf("BaseKeyContribution = %d, want 1", BaseKeyContribution)
	}
}

func TestBaseSaleValue(t *testing.T) {
	if BaseSaleValue != 10 {
		t.Errorf("BaseSaleValue = %d, want 10", BaseSaleValue)
	}
}

func TestUpgradeCostParams(t *testing.T) {
	if UpgradeBaseCost != 50 {
		t.Errorf("UpgradeBaseCost = %d, want 50", UpgradeBaseCost)
	}
	if UpgradeCostMultiplier != 1.5 {
		t.Errorf("UpgradeCostMultiplier = %v, want 1.5", UpgradeCostMultiplier)
	}
}

func TestUpgradeIncrements(t *testing.T) {
	if UpgradeContributionIncrement != 1 {
		t.Errorf("UpgradeContributionIncrement = %d, want 1", UpgradeContributionIncrement)
	}
	if UpgradeValueIncrement != 5 {
		t.Errorf("UpgradeValueIncrement = %d, want 5", UpgradeValueIncrement)
	}
}

func TestCosmeticPrice(t *testing.T) {
	if CosmeticPrice != 100 {
		t.Errorf("CosmeticPrice = %d, want 100", CosmeticPrice)
	}
}

func TestTimingConstants(t *testing.T) {
	if IdleThreshold != 10*time.Second {
		t.Errorf("IdleThreshold = %v, want 10s", IdleThreshold)
	}
	if AutosaveInterval != 30*time.Second {
		t.Errorf("AutosaveInterval = %v, want 30s", AutosaveInterval)
	}
}
