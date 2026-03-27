-- Drop the global UNIQUE constraint on alert_rules.name.
-- Names should be unique per connection, not globally.
-- SQLite doesn't support DROP CONSTRAINT, so we recreate the table.

CREATE TABLE alert_rules_new (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    query TEXT NOT NULL,
    column_name TEXT NOT NULL,
    operator TEXT NOT NULL CHECK (operator IN ('gt', 'gte', 'lt', 'lte', 'eq', 'neq')),
    threshold REAL NOT NULL,
    eval_interval INTEGER NOT NULL DEFAULT 60,
    for_duration INTEGER NOT NULL DEFAULT 0,
    severity TEXT NOT NULL DEFAULT 'warning' CHECK (severity IN ('critical', 'warning', 'info')),
    labels TEXT NOT NULL DEFAULT '{}',
    annotations TEXT NOT NULL DEFAULT '{}',
    channel_ids TEXT NOT NULL DEFAULT '[]',
    connection_id TEXT NOT NULL DEFAULT '',
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO alert_rules_new SELECT id, name, query, column_name, operator, threshold, eval_interval, for_duration, severity, labels, annotations, channel_ids, connection_id, enabled, created_at, updated_at FROM alert_rules;

DROP TABLE alert_rules;

ALTER TABLE alert_rules_new RENAME TO alert_rules;

CREATE UNIQUE INDEX IF NOT EXISTS idx_alert_rules_name_connection ON alert_rules(name, connection_id);
CREATE INDEX IF NOT EXISTS idx_alert_rules_connection_id ON alert_rules(connection_id);
