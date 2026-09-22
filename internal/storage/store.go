package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"nookbuddy/internal/storage/generated"
	"slices"
	"time"
)

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Load(ctx context.Context) (PlayerState, error) {
	queries := generated.New(s.db)

	player, err := queries.GetPlayer(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return PlayerState{}, err
	}

	cosmetics, err := queries.ListCosmetics(ctx)
	if err != nil {
		return PlayerState{}, err
	}

	owned := make([]string, 0, len(cosmetics))
	for _, cosmetic := range cosmetics {
		if cosmetic.Owned == 1 {
			owned = append(owned, cosmetic.CosmeticID)
		}
	}

	return PlayerState{
		ClicksProgress:     int(player.ClicksProgress),
		KeysProgress:       int(player.KeysProgress),
		ProjectsReady:      int(player.ProjectsReady),
		Gold:               int(player.Gold),
		UpgradeClicksLevel: int(player.UpgradeClicksLevel),
		UpgradeKeysLevel:   int(player.UpgradeKeysLevel),
		UpgradeValueLevel:  int(player.UpgradeValueLevel),
		OwnedCosmetics:     owned,
	}, nil
}

func (s *Store) Save(ctx context.Context, state PlayerState) (SaveResult, error) {
	now := time.Now().UTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SaveResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	queries := generated.New(s.db).WithTx(tx)

	err = queries.UpsertPlayer(ctx, generated.UpsertPlayerParams{
		ClicksProgress:     int64(state.ClicksProgress),
		KeysProgress:       int64(state.KeysProgress),
		ProjectsReady:      int64(state.ProjectsReady),
		Gold:               int64(state.Gold),
		UpgradeClicksLevel: int64(state.UpgradeClicksLevel),
		UpgradeKeysLevel:   int64(state.UpgradeKeysLevel),
		UpgradeValueLevel:  int64(state.UpgradeValueLevel),
		UpdatedAt:          now.Format(time.RFC3339),
	})
	if err != nil {
		return SaveResult{}, err
	}

	existing, err := queries.ListCosmetics(ctx)
	if err != nil {
		return SaveResult{}, err
	}
	purchasedAt := make(map[string]sql.NullString, len(existing))
	for _, cosmetic := range existing {
		purchasedAt[cosmetic.CosmeticID] = cosmetic.PurchasedAt
	}

	for _, id := range cosmeticIDsToWrite(state.OwnedCosmetics) {
		owned := slices.Contains(state.OwnedCosmetics, id)
		params := generated.UpsertCosmeticParams{CosmeticID: id}
		if owned {
			params.Owned = 1
			params.PurchasedAt = purchasedAt[id]
			if !params.PurchasedAt.Valid {
				params.PurchasedAt = sql.NullString{String: now.Format(time.RFC3339), Valid: true}
			}
		}
		if err := queries.UpsertCosmetic(ctx, params); err != nil {
			return SaveResult{}, fmt.Errorf("could not persist cosmetic %q: %w", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return SaveResult{}, err
	}

	return SaveResult{OK: true, SavedAt: now}, nil
}

func cosmeticIDsToWrite(owned []string) []string {
	ids := slices.Clone(CosmeticIDs)
	for _, id := range owned {
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids
}
