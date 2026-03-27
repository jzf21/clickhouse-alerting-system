ALTER TABLE notification_channels ADD COLUMN connection_id TEXT DEFAULT NULL;
ALTER TABLE silences ADD COLUMN connection_id TEXT DEFAULT NULL;

CREATE INDEX IF NOT EXISTS idx_notification_channels_connection_id ON notification_channels(connection_id);
CREATE INDEX IF NOT EXISTS idx_silences_connection_id ON silences(connection_id);
CREATE INDEX IF NOT EXISTS idx_alert_rules_connection_id ON alert_rules(connection_id);
