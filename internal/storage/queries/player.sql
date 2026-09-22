-- name: GetPlayer :one
SELECT
    id,
    clicks_progress,
    keys_progress,
    projects_ready,
    gold,
    upgrade_clicks_level,
    upgrade_keys_level,
    upgrade_value_level,
    updated_at
FROM player
WHERE id = 1;

-- name: UpsertPlayer :exec
INSERT INTO player (
    id,
    clicks_progress,
    keys_progress,
    projects_ready,
    gold,
    upgrade_clicks_level,
    upgrade_keys_level,
    upgrade_value_level,
    updated_at
) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET
    clicks_progress = excluded.clicks_progress,
    keys_progress = excluded.keys_progress,
    projects_ready = excluded.projects_ready,
    gold = excluded.gold,
    upgrade_clicks_level = excluded.upgrade_clicks_level,
    upgrade_keys_level = excluded.upgrade_keys_level,
    upgrade_value_level = excluded.upgrade_value_level,
    updated_at = excluded.updated_at;
