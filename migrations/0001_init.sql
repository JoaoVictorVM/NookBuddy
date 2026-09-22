CREATE TABLE player (
    id INTEGER PRIMARY KEY,
    clicks_progress INTEGER NOT NULL DEFAULT 0,
    keys_progress INTEGER NOT NULL DEFAULT 0,
    projects_ready INTEGER NOT NULL DEFAULT 0,
    gold INTEGER NOT NULL DEFAULT 0,
    upgrade_clicks_level INTEGER NOT NULL DEFAULT 0,
    upgrade_keys_level INTEGER NOT NULL DEFAULT 0,
    upgrade_value_level INTEGER NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO player (id) VALUES (1);

CREATE TABLE cosmetics (
    cosmetic_id TEXT PRIMARY KEY CHECK (cosmetic_id IN ('window','bookshelf','flower','painting','rug')),
    owned INTEGER NOT NULL DEFAULT 0,
    purchased_at TEXT
);

INSERT INTO cosmetics (cosmetic_id) VALUES
    ('window'), ('bookshelf'), ('flower'), ('painting'), ('rug');

CREATE TABLE schema_version (
    version INTEGER NOT NULL
);

INSERT INTO schema_version (version) VALUES (1);
