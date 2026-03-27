package evaluator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jozef/clickhouse-alerting-system/internal/connregistry"
	"github.com/jozef/clickhouse-alerting-system/internal/model"
	"github.com/jozef/clickhouse-alerting-system/internal/store"
)

// NotifyFunc is called when an alert fires or resolves.
type NotifyFunc func(ctx context.Context, rule model.AlertRule, state model.AlertState, action Action)

type Evaluator struct {
	store        store.Store
	connRegistry *connregistry.Registry
	queryTimeout time.Duration
	sem          chan struct{}
	notifyFn     NotifyFunc
	repeatIvl    time.Duration

	mu      sync.Mutex
	tickers map[string]*ruleTicker
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

type ruleTicker struct {
	cancel context.CancelFunc
}

func New(st store.Store, registry *connregistry.Registry, queryTimeout time.Duration, maxConcurrent int, repeatInterval time.Duration, notifyFn NotifyFunc) *Evaluator {
	return &Evaluator{
		store:        st,
		connRegistry: registry,
		queryTimeout: queryTimeout,
		sem:          make(chan struct{}, maxConcurrent),
		notifyFn:     notifyFn,
		repeatIvl:    repeatInterval,
		tickers:      make(map[string]*ruleTicker),
		stopCh:       make(chan struct{}),
	}
}

func (e *Evaluator) Start(ctx context.Context) {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.reloadLoop(ctx)
	}()
}

func (e *Evaluator) Stop() {
	close(e.stopCh)
	e.mu.Lock()
	for id, t := range e.tickers {
		t.cancel()
		delete(e.tickers, id)
	}
	e.mu.Unlock()
	e.wg.Wait()
}

func (e *Evaluator) reloadLoop(ctx context.Context) {
	e.reload(ctx)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-e.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.reload(ctx)
		}
	}
}

func (e *Evaluator) reload(ctx context.Context) {
	rules, err := e.store.ListEnabledRules(ctx)
	if err != nil {
		slog.Error("failed to load rules", "error", err)
		return
	}

	wanted := make(map[string]model.AlertRule, len(rules))
	for _, r := range rules {
		wanted[r.ID] = r
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Stop tickers for rules that no longer exist or are disabled
	for id, t := range e.tickers {
		if _, ok := wanted[id]; !ok {
			t.cancel()
			delete(e.tickers, id)
		}
	}

	// Start tickers for new rules
	for _, rule := range rules {
		if _, ok := e.tickers[rule.ID]; !ok {
			ctx2, cancel := context.WithCancel(ctx)
			e.tickers[rule.ID] = &ruleTicker{cancel: cancel}
			e.wg.Add(1)
			go func(r model.AlertRule) {
				defer e.wg.Done()
				e.runRule(ctx2, r)
			}(rule)
		}
	}

	slog.Debug("evaluator reload", "active_rules", len(e.tickers))
}

func (e *Evaluator) runRule(ctx context.Context, rule model.AlertRule) {
	interval := time.Duration(rule.EvalInterval) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}

	// Evaluate immediately, then on interval
	e.evaluateRule(ctx, rule)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			// Re-fetch rule in case it was updated
			updated, err := e.store.GetRule(ctx, rule.ID)
			if err != nil {
				slog.Error("rule fetch failed, stopping", "rule_id", rule.ID, "error", err)
				return
			}
			if !updated.Enabled {
				return
			}
			e.evaluateRule(ctx, updated)
		}
	}
}

func (e *Evaluator) evaluateRule(ctx context.Context, rule model.AlertRule) {
	// Acquire semaphore
	select {
	case e.sem <- struct{}{}:
	case <-ctx.Done():
		return
	}
	defer func() { <-e.sem }()

	value, err := e.runQuery(ctx, rule)
	if err != nil {
		slog.Error("query failed", "rule", rule.Name, "error", err)
		return // Query errors don't change state
	}

	state, err := e.store.GetAlertState(ctx, rule.ID)
	if err != nil {
		slog.Error("get state failed", "rule", rule.Name, "error", err)
		return
	}

	conditionMet := EvaluateCondition(value, rule.Operator, rule.Threshold)
	forDur := time.Duration(rule.ForDuration) * time.Second
	now := time.Now().UTC()

	result := Transition(state, conditionMet, forDur, now, value)

	if err := e.store.UpsertAlertState(ctx, result.NewState); err != nil {
		slog.Error("upsert state failed", "rule", rule.Name, "error", err)
		return
	}

	switch result.Action {
	case ActionFiring:
		e.onFiring(ctx, rule, result.NewState, value)
	case ActionResolved:
		e.onResolved(ctx, rule, result.NewState, value)
	default:
		// Check for re-notification of still-firing alerts
		if result.NewState.State == "firing" && e.shouldRenotify(result.NewState) {
			e.onFiring(ctx, rule, result.NewState, value)
		}
	}

	slog.Debug("evaluated", "rule", rule.Name, "value", value, "condition", conditionMet, "state", result.NewState.State)
}

func (e *Evaluator) shouldRenotify(state model.AlertState) bool {
	if state.LastNotifiedAt == nil || e.repeatIvl <= 0 {
		return false
	}
	return time.Since(*state.LastNotifiedAt) >= e.repeatIvl
}

func (e *Evaluator) onFiring(ctx context.Context, rule model.AlertRule, state model.AlertState, value float64) {
	event := model.AlertEvent{
		ID:          uuid.New().String(),
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		State:       "firing",
		Value:       value,
		Severity:    rule.Severity,
		Labels:      rule.Labels,
		Annotations: rule.Annotations,
		CreatedAt:   time.Now().UTC(),
	}
	if err := e.store.CreateEvent(ctx, event); err != nil {
		slog.Error("create event failed", "error", err)
	}

	now := time.Now().UTC()
	state.LastNotifiedAt = &now
	if err := e.store.UpsertAlertState(ctx, state); err != nil {
		slog.Error("update notified_at failed", "error", err)
	}

	if e.notifyFn != nil {
		e.notifyFn(ctx, rule, state, ActionFiring)
	}
}

func (e *Evaluator) onResolved(ctx context.Context, rule model.AlertRule, state model.AlertState, value float64) {
	event := model.AlertEvent{
		ID:          uuid.New().String(),
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		State:       "resolved",
		Value:       value,
		Severity:    rule.Severity,
		Labels:      rule.Labels,
		Annotations: rule.Annotations,
		CreatedAt:   time.Now().UTC(),
	}
	if err := e.store.CreateEvent(ctx, event); err != nil {
		slog.Error("create event failed", "error", err)
	}

	if e.notifyFn != nil {
		e.notifyFn(ctx, rule, state, ActionResolved)
	}
}

func (e *Evaluator) runQuery(ctx context.Context, rule model.AlertRule) (float64, error) {
	if rule.ConnectionID == "" {
		return 0, fmt.Errorf("rule %q has no connection_id configured", rule.Name)
	}

	db, err := e.connRegistry.Get(ctx, rule.ConnectionID)
	if err != nil {
		return 0, fmt.Errorf("getting connection for rule %q: %w", rule.Name, err)
	}

	queryCtx, cancel := context.WithTimeout(ctx, e.queryTimeout)
	defer cancel()

	var value float64
	row := db.QueryRowContext(queryCtx, rule.Query)
	if err := row.Scan(&value); err != nil {
		return 0, fmt.Errorf("scanning query result for column %q: %w", rule.Column, err)
	}
	return value, nil
}
