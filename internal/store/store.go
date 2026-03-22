package store

import (
	"context"

	"github.com/jozef/clickhouse-alerting-system/internal/model"
)

type Store interface {
	// Alert Rules
	ListRules(ctx context.Context) ([]model.AlertRule, error)
	GetRule(ctx context.Context, id string) (model.AlertRule, error)
	CreateRule(ctx context.Context, rule model.AlertRule) error
	UpdateRule(ctx context.Context, rule model.AlertRule) error
	DeleteRule(ctx context.Context, id string) error
	ListEnabledRules(ctx context.Context) ([]model.AlertRule, error)

	// Alert States
	GetAlertState(ctx context.Context, ruleID string) (model.AlertState, error)
	UpsertAlertState(ctx context.Context, state model.AlertState) error
	ListAlertStates(ctx context.Context) ([]model.AlertWithRule, error)
	DeleteAlertState(ctx context.Context, ruleID string) error

	// Alert Events
	CreateEvent(ctx context.Context, event model.AlertEvent) error
	ListEvents(ctx context.Context, ruleID string, limit, offset int) ([]model.AlertEvent, error)

	// Silences
	ListSilences(ctx context.Context) ([]model.Silence, error)
	GetSilence(ctx context.Context, id string) (model.Silence, error)
	CreateSilence(ctx context.Context, silence model.Silence) error
	DeleteSilence(ctx context.Context, id string) error
	ListActiveSilences(ctx context.Context) ([]model.Silence, error)

	// Notification Channels
	ListChannels(ctx context.Context) ([]model.NotificationChannel, error)
	GetChannel(ctx context.Context, id string) (model.NotificationChannel, error)
	CreateChannel(ctx context.Context, ch model.NotificationChannel) error
	UpdateChannel(ctx context.Context, ch model.NotificationChannel) error
	DeleteChannel(ctx context.Context, id string) error
}
