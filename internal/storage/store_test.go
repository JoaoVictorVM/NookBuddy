package storage

import (
	"context"
	"errors"
	"nookbuddy/migrations"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func populatedState() PlayerState {
	return PlayerState{
		ClicksProgress:     37,
		KeysProgress:       214,
		ProjectsReady:      6,
		Gold:               980,
		UpgradeClicksLevel: 3,
		UpgradeKeysLevel:   2,
		UpgradeValueLevel:  5,
		OwnedCosmetics:     []string{"bookshelf", "flower", "rug"},
	}
}

func TestStore_SaveThenLoadRoundTrip(t *testing.T) {
	path := tempDBPath(t)
	ctx := context.Background()
	want := populatedState()

	store, _ := openTestStore(t, path, migrations.FS)
	result, err := store.Save(ctx, want)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !result.OK {
		t.Error("SaveResult.OK = false, want true")
	}
	if result.SavedAt.IsZero() {
		t.Error("SaveResult.SavedAt is zero, want the save timestamp")
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reopened, notice := openTestStore(t, path, migrations.FS)
	if notice != NoticeNone {
		t.Fatalf("notice = %v, want NoticeNone", notice)
	}
	got, err := reopened.Load(ctx)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	slices.Sort(got.OwnedCosmetics)
	slices.Sort(want.OwnedCosmetics)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("loaded state = %+v, want %+v", got, want)
	}
}

func TestStore_SaveIsAtomic(t *testing.T) {
	ctx := context.Background()
	store, _ := openTestStore(t, tempDBPath(t), migrations.FS)

	original := populatedState()
	if _, err := store.Save(ctx, original); err != nil {
		t.Fatalf("initial Save: %v", err)
	}

	broken := original
	broken.Gold = 1
	broken.ProjectsReady = 99
	broken.OwnedCosmetics = append(slices.Clone(original.OwnedCosmetics), "spaceship")

	if _, err := store.Save(ctx, broken); err == nil {
		t.Fatal("Save with an unknown cosmetic succeeded, want an error")
	}

	got, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Gold != original.Gold {
		t.Errorf("Gold = %d, want %d (the player write must roll back)", got.Gold, original.Gold)
	}
	if got.ProjectsReady != original.ProjectsReady {
		t.Errorf("ProjectsReady = %d, want %d (the player write must roll back)", got.ProjectsReady, original.ProjectsReady)
	}
	slices.Sort(got.OwnedCosmetics)
	if !reflect.DeepEqual(got.OwnedCosmetics, original.OwnedCosmetics) {
		t.Errorf("OwnedCosmetics = %v, want %v", got.OwnedCosmetics, original.OwnedCosmetics)
	}
}

func TestStore_SaveRespectsContextCancellation(t *testing.T) {
	store, _ := openTestStore(t, tempDBPath(t), migrations.FS)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := store.Save(ctx, populatedState()); !errors.Is(err, context.Canceled) {
		t.Fatalf("Save error = %v, want context.Canceled", err)
	}

	got, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Gold != 0 {
		t.Errorf("Gold = %d, want 0 (a canceled save must not write)", got.Gold)
	}
}

func TestStore_UnknownCosmeticIDIsRejected(t *testing.T) {
	ctx := context.Background()
	store, _ := openTestStore(t, tempDBPath(t), migrations.FS)

	state := populatedState()
	state.OwnedCosmetics = append(state.OwnedCosmetics, "spaceship")

	_, err := store.Save(ctx, state)
	if err == nil {
		t.Fatal("Save succeeded with an unknown cosmetic ID, want an error")
	}
	if !strings.Contains(err.Error(), "spaceship") {
		t.Errorf("error = %q, want it to name the rejected cosmetic", err)
	}
}

func TestStore_LoadFeedsProgressAndUpgradeConsumers(t *testing.T) {
	ctx := context.Background()
	store, _ := openTestStore(t, tempDBPath(t), migrations.FS)

	want := populatedState()
	if _, err := store.Save(ctx, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	bars := struct{ Clicks, Keys, ProjectsReady int }{
		got.ClicksProgress, got.KeysProgress, got.ProjectsReady,
	}
	if bars.Clicks != want.ClicksProgress || bars.Keys != want.KeysProgress || bars.ProjectsReady != want.ProjectsReady {
		t.Errorf("bar seed = %+v, want clicks=%d keys=%d ready=%d",
			bars, want.ClicksProgress, want.KeysProgress, want.ProjectsReady)
	}

	levels := struct{ Clicks, Keys, Value int }{
		got.UpgradeClicksLevel, got.UpgradeKeysLevel, got.UpgradeValueLevel,
	}
	if levels.Clicks != want.UpgradeClicksLevel || levels.Keys != want.UpgradeKeysLevel || levels.Value != want.UpgradeValueLevel {
		t.Errorf("upgrade seed = %+v, want %d/%d/%d",
			levels, want.UpgradeClicksLevel, want.UpgradeKeysLevel, want.UpgradeValueLevel)
	}
}

func TestStore_LoadExposesGoldAndCosmeticsForRendering(t *testing.T) {
	ctx := context.Background()
	store, _ := openTestStore(t, tempDBPath(t), migrations.FS)

	want := populatedState()
	if _, err := store.Save(ctx, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Gold != want.Gold {
		t.Errorf("Gold = %d, want %d", got.Gold, want.Gold)
	}
	slices.Sort(got.OwnedCosmetics)
	slices.Sort(want.OwnedCosmetics)
	if !reflect.DeepEqual(got.OwnedCosmetics, want.OwnedCosmetics) {
		t.Errorf("OwnedCosmetics = %v, want %v", got.OwnedCosmetics, want.OwnedCosmetics)
	}
}
