package evaluator

import (
	"time"

	"github.com/jozef/clickhouse-alerting-system/internal/model"
)

// Action describes what happened after a state transition.
type Action string

const (
	ActionNone     Action = ""
	ActionFiring   Action = "firing"
	ActionResolved Action = "resolved"
)

// EvalResult holds the result of a single evaluation tick.
type EvalResult struct {
	NewState model.AlertState
	Action   Action
}

// EvaluateCondition returns true if the value breaches the threshold.
func EvaluateCondition(value float64, operator string, threshold float64) bool {
	switch operator {
	case "gt":
		return value > threshold
	case "gte":
		return value >= threshold
	case "lt":
		return value < threshold
	case "lte":
		return value <= threshold
	case "eq":
		return value == threshold
	case "neq":
		return value != threshold
	default:
		return false
	}
}

// Transition computes the next alert state given the current state and evaluation result.
// This is a pure function for testability.
func Transition(current model.AlertState, conditionMet bool, forDuration time.Duration, now time.Time, value float64) EvalResult {
	next := current
	next.LastEvalAt = &now
	next.LastEvalValue = &value
	action := ActionNone

	switch current.State {
	case "inactive", "":
		if conditionMet {
			if forDuration <= 0 {
				// No for_duration, go straight to firing
				next.State = "firing"
				next.FiringSince = &now
				next.PendingSince = nil
				next.ResolvedAt = nil
				action = ActionFiring
			} else {
				next.State = "pending"
				next.PendingSince = &now
				next.ResolvedAt = nil
			}
		}

	case "pending":
		if !conditionMet {
			next.State = "inactive"
			next.PendingSince = nil
		} else if current.PendingSince != nil && now.Sub(*current.PendingSince) >= forDuration {
			next.State = "firing"
			next.FiringSince = &now
			next.PendingSince = nil
			action = ActionFiring
		}

	case "firing":
		if !conditionMet {
			next.State = "inactive"
			resolved := now
			next.ResolvedAt = &resolved
			next.FiringSince = nil
			next.PendingSince = nil
			action = ActionResolved
		}
	}

	return EvalResult{NewState: next, Action: action}
}
