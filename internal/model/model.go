package model

import (
	"encoding/json"
	"time"
)

type AlertRule struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Query        string          `json:"query"`
	Column       string          `json:"column"`
	Operator     string          `json:"operator"` // gt, gte, lt, lte, eq, neq
	Threshold    float64         `json:"threshold"`
	EvalInterval int             `json:"eval_interval"` // seconds
	ForDuration  int             `json:"for_duration"`   // seconds
	Severity     string          `json:"severity"`       // critical, warning, info
	Labels       json.RawMessage `json:"labels"`
	Annotations  json.RawMessage `json:"annotations"`
	ChannelIDs   json.RawMessage `json:"channel_ids"`
	Enabled      bool            `json:"enabled"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type AlertState struct {
	RuleID        string    `json:"rule_id"`
	State         string    `json:"state"` // inactive, pending, firing
	PendingSince  *time.Time `json:"pending_since,omitempty"`
	FiringSince   *time.Time `json:"firing_since,omitempty"`
	LastEvalAt    *time.Time `json:"last_eval_at,omitempty"`
	LastEvalValue *float64   `json:"last_eval_value,omitempty"`
	LastNotifiedAt *time.Time `json:"last_notified_at,omitempty"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
}

type AlertEvent struct {
	ID          string          `json:"id"`
	RuleID      string          `json:"rule_id"`
	RuleName    string          `json:"rule_name"`
	State       string          `json:"state"` // firing, resolved
	Value       float64         `json:"value"`
	Severity    string          `json:"severity"`
	Labels      json.RawMessage `json:"labels"`
	Annotations json.RawMessage `json:"annotations"`
	CreatedAt   time.Time       `json:"created_at"`
}

type Silence struct {
	ID        string          `json:"id"`
	Matchers  json.RawMessage `json:"matchers"` // [{label, value, is_regex}]
	Comment   string          `json:"comment"`
	CreatedBy string          `json:"created_by"`
	StartsAt  time.Time       `json:"starts_at"`
	EndsAt    time.Time       `json:"ends_at"`
	CreatedAt time.Time       `json:"created_at"`
}

type LabelMatcher struct {
	Label   string `json:"label"`
	Value   string `json:"value"`
	IsRegex bool   `json:"is_regex"`
}

type NotificationChannel struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"` // slack, webhook
	Config    json.RawMessage `json:"config"`
	Enabled   bool            `json:"enabled"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel,omitempty"`
	Username   string `json:"username,omitempty"`
}

type WebhookConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// AlertWithRule combines alert state with its rule for display.
type AlertWithRule struct {
	AlertState
	RuleName    string          `json:"rule_name"`
	Severity    string          `json:"severity"`
	Labels      json.RawMessage `json:"labels"`
	Annotations json.RawMessage `json:"annotations"`
}
