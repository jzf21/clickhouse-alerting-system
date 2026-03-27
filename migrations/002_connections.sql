CREATE TABLE IF NOT EXISTS clickhouse_connections (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    host TEXT NOT NULL,
    port INTEGER NOT NULL DEFAULT 9000,
    database_name TEXT NOT NULL DEFAULT 'default',
    username TEXT NOT NULL DEFAULT 'default',
    password TEXT NOT NULL DEFAULT '',
    secure INTEGER NOT NULL DEFAULT 0,
    max_open_conns INTEGER NOT NULL DEFAULT 5,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

ALTER TABLE alert_rules ADD COLUMN connection_id TEXT NOT NULL DEFAULT '';
