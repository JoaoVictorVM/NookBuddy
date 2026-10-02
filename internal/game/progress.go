package game

import (
	"nookbuddy/internal/config"
	"nookbuddy/internal/input"
	"nookbuddy/internal/storage"
)

func ApplyEvent(state storage.PlayerState, event input.Event) storage.PlayerState {
	switch event.Type {
	case input.Click:
		state.ClicksProgress = min(state.ClicksProgress+ClickContribution(state.UpgradeClicksLevel), config.ClicksBarLimit)
	case input.Key:
		state.KeysProgress = min(state.KeysProgress+KeyContribution(state.UpgradeKeysLevel), config.KeysBarLimit)
	default:
		return state
	}

	if IsProjectComplete(state) {
		state.ProjectsReady++
		state.ClicksProgress, state.KeysProgress = 0, 0
	}
	return state
}

func IsProjectComplete(state storage.PlayerState) bool {
	return state.ClicksProgress >= config.ClicksBarLimit && state.KeysProgress >= config.KeysBarLimit
}
