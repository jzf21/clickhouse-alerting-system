package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	"github.com/jozef/clickhouse-alerting-system/internal/model"
	_ "modernc.org/sqlite"
)

// MigrationsFS is set from main.go with the embedded migrations.
var MigrationsFS embed.FS

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite is single-writer
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) migrate() error {
	data, err := MigrationsFS.ReadFile("migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("reading migration: %w", err)
	}
	_, err = s.db.Exec(string(data))
	return err
}

// --- Alert Rules ---

func (s *SQLiteStore) ListRules(ctx context.Context) ([]model.AlertRule, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, query, column_name, operator, threshold, eval_interval, for_duration, severity, labels, annotations, channel_ids, enabled, created_at, updated_at FROM alert_rules ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRules(rows)
}

func (s *SQLiteStore) ListEnabledRules(ctx context.Context) ([]model.AlertRule, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, query, column_name, operator, threshold, eval_interval, for_duration, severity, labels, annotations, channel_ids, enabled, created_at, updated_at FROM alert_rules WHERE enabled = 1 ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRules(rows)
}

func (s *SQLiteStore) GetRule(ctx context.Context, id string) (model.AlertRule, error) {
	var r model.AlertRule
	err := s.db.QueryRowContext(ctx, `SELECT id, name, query, column_name, operator, threshold, eval_interval, for_duration, severity, labels, annotations, channel_ids, enabled, created_at, updated_at FROM alert_rules WHERE id = ?`, id).Scan(
		&r.ID, &r.Name, &r.Query, &r.Column, &r.Operator, &r.Threshold,
		&r.EvalInterval, &r.ForDuration, &r.Severity, &r.Labels, &r.Annotations,
		&r.ChannelIDs, &r.Enabled, &r.CreatedAt, &r.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return r, fmt.Errorf("rule not found: %s", id)
	}
	return r, err
}

func (s *SQLiteStore) CreateRule(ctx context.Context, rule model.AlertRule) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO alert_rules (id, name, query, column_name, operator, threshold, eval_interval, for_duration, severity, labels, annotations, channel_ids, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.ID, rule.Name, rule.Query, rule.Column, rule.Operator, rule.Threshold,
		rule.EvalInterval, rule.ForDuration, rule.Severity, rule.Labels, rule.Annotations,
		rule.ChannelIDs, rule.Enabled, rule.CreatedAt, rule.UpdatedAt,
	)
	return err
}

func (s *SQLiteStore) UpdateRule(ctx context.Context, rule model.AlertRule) error {
	res, err := s.db.ExecContext(ctx, `UPDATE alert_rules SET name=?, query=?, column_name=?, operator=?, threshold=?, eval_interval=?, for_duration=?, severity=?, labels=?, annotations=?, channel_ids=?, enabled=?, updated_at=? WHERE id=?`,
		rule.Name, rule.Query, rule.Column, rule.Operator, rule.Threshold,
		rule.EvalInterval, rule.ForDuration, rule.Severity, rule.Labels, rule.Annotations,
		rule.ChannelIDs, rule.Enabled, rule.UpdatedAt, rule.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("rule not found: %s", rule.ID)
	}
	return nil
}

func (s *SQLiteStore) DeleteRule(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM alert_rules WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("rule not found: %s", id)
	}
	return nil
}

func scanRules(rows *sql.Rows) ([]model.AlertRule, error) {
	var rules []model.AlertRule
	for rows.Next() {
		var r model.AlertRule
		if err := rows.Scan(
			&r.ID, &r.Name, &r.Query, &r.Column, &r.Operator, &r.Threshold,
			&r.EvalInterval, &r.ForDuration, &r.Severity, &r.Labels, &r.Annotations,
			&r.ChannelIDs, &r.Enabled, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

// --- Alert States ---

func (s *SQLiteStore) GetAlertState(ctx context.Context, ruleID string) (model.AlertState, error) {
	var st model.AlertState
	err := s.db.QueryRowContext(ctx, `SELECT rule_id, state, pending_since, firing_since, last_eval_at, last_eval_value, last_notified_at, resolved_at FROM alert_states WHERE rule_id = ?`, ruleID).Scan(
		&st.RuleID, &st.State, &st.PendingSince, &st.FiringSince,
		&st.LastEvalAt, &st.LastEvalValue, &st.LastNotifiedAt, &st.ResolvedAt,
	)
	if err == sql.ErrNoRows {
		return model.AlertState{RuleID: ruleID, State: "inactive"}, nil
	}
	return st, err
}

func (s *SQLiteStore) UpsertAlertState(ctx context.Context, state model.AlertState) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO alert_states (rule_id, state, pending_since, firing_since, last_eval_at, last_eval_value, last_notified_at, resolved_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(rule_id) DO UPDATE SET state=excluded.state, pending_since=excluded.pending_since, firing_since=excluded.firing_since, last_eval_at=excluded.last_eval_at, last_eval_value=excluded.last_eval_value, last_notified_at=excluded.last_notified_at, resolved_at=excluded.resolved_at`,
		state.RuleID, state.State, state.PendingSince, state.FiringSince,
		state.LastEvalAt, state.LastEvalValue, state.LastNotifiedAt, state.ResolvedAt,
	)
	return err
}

func (s *SQLiteStore) ListAlertStates(ctx context.Context) ([]model.AlertWithRule, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT s.rule_id, s.state, s.pending_since, s.firing_since, s.last_eval_at, s.last_eval_value, s.last_notified_at, s.resolved_at, r.name, r.severity, r.labels, r.annotations FROM alert_states s JOIN alert_rules r ON s.rule_id = r.id ORDER BY CASE s.state WHEN 'firing' THEN 0 WHEN 'pending' THEN 1 ELSE 2 END, r.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.AlertWithRule
	for rows.Next() {
		var a model.AlertWithRule
		if err := rows.Scan(
			&a.RuleID, &a.State, &a.PendingSince, &a.FiringSince,
			&a.LastEvalAt, &a.LastEvalValue, &a.LastNotifiedAt, &a.ResolvedAt,
			&a.RuleName, &a.Severity, &a.Labels, &a.Annotations,
		); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) DeleteAlertState(ctx context.Context, ruleID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM alert_states WHERE rule_id = ?`, ruleID)
	return err
}

// --- Alert Events ---

func (s *SQLiteStore) CreateEvent(ctx context.Context, event model.AlertEvent) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO alert_events (id, rule_id, rule_name, state, value, severity, labels, annotations, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.RuleID, event.RuleName, event.State, event.Value,
		event.Severity, event.Labels, event.Annotations, event.CreatedAt,
	)
	return err
}

func (s *SQLiteStore) ListEvents(ctx context.Context, ruleID string, limit, offset int) ([]model.AlertEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows *sql.Rows
	var err error
	if ruleID != "" {
		rows, err = s.db.QueryContext(ctx, `SELECT id, rule_id, rule_name, state, value, severity, labels, annotations, created_at FROM alert_events WHERE rule_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, ruleID, limit, offset)
	} else {
		rows, err = s.db.QueryContext(ctx, `SELECT id, rule_id, rule_name, state, value, severity, labels, annotations, created_at FROM alert_events ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []model.AlertEvent
	for rows.Next() {
		var e model.AlertEvent
		if err := rows.Scan(&e.ID, &e.RuleID, &e.RuleName, &e.State, &e.Value, &e.Severity, &e.Labels, &e.Annotations, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// --- Silences ---

func (s *SQLiteStore) ListSilences(ctx context.Context) ([]model.Silence, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, matchers, comment, created_by, starts_at, ends_at, created_at FROM silences ORDER BY ends_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSilences(rows)
}

func (s *SQLiteStore) ListActiveSilences(ctx context.Context) ([]model.Silence, error) {
	now := time.Now().UTC()
	rows, err := s.db.QueryContext(ctx, `SELECT id, matchers, comment, created_by, starts_at, ends_at, created_at FROM silences WHERE starts_at <= ? AND ends_at > ? ORDER BY ends_at DESC`, now, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSilences(rows)
}

func (s *SQLiteStore) GetSilence(ctx context.Context, id string) (model.Silence, error) {
	var si model.Silence
	err := s.db.QueryRowContext(ctx, `SELECT id, matchers, comment, created_by, starts_at, ends_at, created_at FROM silences WHERE id = ?`, id).Scan(
		&si.ID, &si.Matchers, &si.Comment, &si.CreatedBy, &si.StartsAt, &si.EndsAt, &si.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return si, fmt.Errorf("silence not found: %s", id)
	}
	return si, err
}

func (s *SQLiteStore) CreateSilence(ctx context.Context, silence model.Silence) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO silences (id, matchers, comment, created_by, starts_at, ends_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		silence.ID, silence.Matchers, silence.Comment, silence.CreatedBy, silence.StartsAt, silence.EndsAt, silence.CreatedAt,
	)
	return err
}

func (s *SQLiteStore) DeleteSilence(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM silences WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("silence not found: %s", id)
	}
	return nil
}

func scanSilences(rows *sql.Rows) ([]model.Silence, error) {
	var silences []model.Silence
	for rows.Next() {
		var si model.Silence
		if err := rows.Scan(&si.ID, &si.Matchers, &si.Comment, &si.CreatedBy, &si.StartsAt, &si.EndsAt, &si.CreatedAt); err != nil {
			return nil, err
		}
		silences = append(silences, si)
	}
	return silences, rows.Err()
}

// --- Notification Channels ---

func (s *SQLiteStore) ListChannels(ctx context.Context) ([]model.NotificationChannel, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, type, config, enabled, created_at, updated_at FROM notification_channels ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var channels []model.NotificationChannel
	for rows.Next() {
		var ch model.NotificationChannel
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Config, &ch.Enabled, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}
	return channels, rows.Err()
}

func (s *SQLiteStore) GetChannel(ctx context.Context, id string) (model.NotificationChannel, error) {
	var ch model.NotificationChannel
	err := s.db.QueryRowContext(ctx, `SELECT id, name, type, config, enabled, created_at, updated_at FROM notification_channels WHERE id = ?`, id).Scan(
		&ch.ID, &ch.Name, &ch.Type, &ch.Config, &ch.Enabled, &ch.CreatedAt, &ch.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return ch, fmt.Errorf("channel not found: %s", id)
	}
	return ch, err
}

func (s *SQLiteStore) CreateChannel(ctx context.Context, ch model.NotificationChannel) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO notification_channels (id, name, type, config, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ch.ID, ch.Name, ch.Type, ch.Config, ch.Enabled, ch.CreatedAt, ch.UpdatedAt,
	)
	return err
}

func (s *SQLiteStore) UpdateChannel(ctx context.Context, ch model.NotificationChannel) error {
	res, err := s.db.ExecContext(ctx, `UPDATE notification_channels SET name=?, type=?, config=?, enabled=?, updated_at=? WHERE id=?`,
		ch.Name, ch.Type, ch.Config, ch.Enabled, ch.UpdatedAt, ch.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("channel not found: %s", ch.ID)
	}
	return nil
}

func (s *SQLiteStore) DeleteChannel(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM notification_channels WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("channel not found: %s", id)
	}
	return nil
}
