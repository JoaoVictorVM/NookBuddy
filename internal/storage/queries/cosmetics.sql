-- name: ListCosmetics :many
SELECT cosmetic_id, owned, purchased_at
FROM cosmetics
ORDER BY cosmetic_id;

-- name: UpsertCosmetic :exec
INSERT INTO cosmetics (cosmetic_id, owned, purchased_at)
VALUES (?, ?, ?)
ON CONFLICT (cosmetic_id) DO UPDATE SET
    owned = excluded.owned,
    purchased_at = excluded.purchased_at;
