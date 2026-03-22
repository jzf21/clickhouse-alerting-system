package notifier

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"

	"github.com/jozef/clickhouse-alerting-system/internal/evaluator"
	"github.com/jozef/clickhouse-alerting-system/internal/model"
	"github.com/jozef/clickhouse-alerting-system/internal/store"
)

// Sender sends a notification for a given alert using channel-specific config.
type Sender interface {
	Send(ctx context.Context, channelConfig json.RawMessage, alert Alert) error
}

// Alert is the notification payload.
type Alert struct {
	RuleName    string            `json:"rule_name"`
	State       string            `json:"state"` // firing or resolved
	Value       float64           `json:"value"`
	Threshold   float64           `json:"threshold"`
	Operator    string            `json:"operator"`
	Severity    string            `json:"severity"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

type Dispatcher struct {
	store   store.Store
	senders map[string]Sender
}

func NewDispatcher(st store.Store) *Dispatcher {
	return &Dispatcher{
		store: st,
		senders: map[string]Sender{
			"slack":   &SlackSender{},
			"webhook": &WebhookSender{},
		},
	}
}

// NotifyFunc returns a function compatible with evaluator.NotifyFunc.
func (d *Dispatcher) NotifyFunc() evaluator.NotifyFunc {
	return func(ctx context.Context, rule model.AlertRule, state model.AlertState, action evaluator.Action) {
		d.dispatch(ctx, rule, state, action)
	}
}

func (d *Dispatcher) dispatch(ctx context.Context, rule model.AlertRule, state model.AlertState, action evaluator.Action) {
	silenced, err := d.isSilenced(ctx, rule)
	if err != nil {
		slog.Error("checking silences failed", "error", err)
	}
	if silenced {
		slog.Info("alert silenced", "rule", rule.Name)
		return
	}

	var value float64
	if state.LastEvalValue != nil {
		value = *state.LastEvalValue
	}

	alert := Alert{
		RuleName:    rule.Name,
		State:       string(action),
		Value:       value,
		Threshold:   rule.Threshold,
		Operator:    rule.Operator,
		Severity:    rule.Severity,
		Labels:      parseJSONMap(rule.Labels),
		Annotations: parseJSONMap(rule.Annotations),
	}

	var channelIDs []string
	if err := json.Unmarshal(rule.ChannelIDs, &channelIDs); err != nil {
		slog.Error("parsing channel_ids", "rule", rule.Name, "error", err)
		return
	}

	for _, chID := range channelIDs {
		ch, err := d.store.GetChannel(ctx, chID)
		if err != nil {
			slog.Error("get channel failed", "channel_id", chID, "error", err)
			continue
		}
		if !ch.Enabled {
			continue
		}
		sender, ok := d.senders[ch.Type]
		if !ok {
			slog.Error("unknown channel type", "type", ch.Type)
			continue
		}
		if err := sender.Send(ctx, ch.Config, alert); err != nil {
			slog.Error("send notification failed", "channel", ch.Name, "type", ch.Type, "error", err)
		} else {
			slog.Info("notification sent", "channel", ch.Name, "rule", rule.Name, "state", string(action))
		}
	}
}

func (d *Dispatcher) isSilenced(ctx context.Context, rule model.AlertRule) (bool, error) {
	silences, err := d.store.ListActiveSilences(ctx)
	if err != nil {
		return false, err
	}
	ruleLabels := parseJSONMap(rule.Labels)
	for _, silence := range silences {
		var matchers []model.LabelMatcher
		if err := json.Unmarshal(silence.Matchers, &matchers); err != nil {
			slog.Error("parsing silence matchers", "silence_id", silence.ID, "error", err)
			continue
		}
		if matchesAll(ruleLabels, matchers) {
			return true, nil
		}
	}
	return false, nil
}

func matchesAll(labels map[string]string, matchers []model.LabelMatcher) bool {
	if len(matchers) == 0 {
		return false
	}
	for _, m := range matchers {
		labelVal, ok := labels[m.Label]
		if !ok {
			return false
		}
		if m.IsRegex {
			re, err := regexp.Compile(m.Value)
			if err != nil {
				return false
			}
			if !re.MatchString(labelVal) {
				return false
			}
		} else if labelVal != m.Value {
			return false
		}
	}
	return true
}

func parseJSONMap(data json.RawMessage) map[string]string {
	m := make(map[string]string)
	if len(data) > 0 {
		json.Unmarshal(data, &m)
	}
	return m
}

// SendTest sends a test notification through a channel.
func (d *Dispatcher) SendTest(ctx context.Context, ch model.NotificationChannel) error {
	sender, ok := d.senders[ch.Type]
	if !ok {
		return nil
	}
	alert := Alert{
		RuleName:  "Test Alert",
		State:     "firing",
		Value:     42,
		Threshold: 10,
		Operator:  "gt",
		Severity:  "info",
		Labels:    map[string]string{"source": "test"},
	}
	return sender.Send(ctx, ch.Config, alert)
}
