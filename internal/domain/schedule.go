package domain

import (
	"fmt"
	"time"
)

type ScheduleRule struct {
	ID           ID
	InstrumentID ID
	ItemID       ID
	PeriodDays   int
	WarningDays  int
	EffectiveAt  time.Time
	SupersedesID ID
}

type ScheduleApplication struct {
	RuleID          ID
	RuleEffectiveAt time.Time
	DueAt           time.Time
	Reason          string
}

func (r ScheduleRule) Validate() error {
	if r.InstrumentID.Empty() || r.ItemID.Empty() || r.PeriodDays <= 0 {
		return NewValidationError("schedule_rule", "instrument, item and period are required")
	}
	return nil
}

func (r ScheduleRule) NextDue(lastQualifiedAt time.Time) (time.Time, error) {
	if err := r.Validate(); err != nil {
		return time.Time{}, err
	}
	anchor := lastQualifiedAt
	if anchor.Before(r.EffectiveAt) {
		anchor = r.EffectiveAt
	}
	return anchor.UTC().AddDate(0, 0, r.PeriodDays), nil
}

func (r ScheduleRule) Evaluate(plan CalibrationPlan, lastQualifiedAt, asOf time.Time) (ScheduleApplication, bool, error) {
	if err := r.Validate(); err != nil {
		return ScheduleApplication{}, false, err
	}
	if r.ID.Empty() {
		return ScheduleApplication{}, false, NewValidationError("schedule_rule", "rule id is required for plan application")
	}
	if plan.InstrumentID != r.InstrumentID || plan.ItemID != r.ItemID {
		return ScheduleApplication{}, false, NewValidationError("schedule_rule", "rule does not target this plan")
	}
	// A future-dated rule must not take effect before its effective date, so a
	// manual reschedule made before that date is preserved.
	if asOf.Before(r.EffectiveAt) {
		return ScheduleApplication{}, false, nil
	}
	// Re-applying the rule already active on the plan would only drift the
	// version and due date, so treat it as a no-op.
	if plan.ScheduleRuleID == r.ID {
		return ScheduleApplication{}, false, nil
	}
	dueAt, err := r.NextDue(lastQualifiedAt)
	if err != nil {
		return ScheduleApplication{}, false, err
	}
	if plan.DueAt.Equal(dueAt) {
		return ScheduleApplication{}, false, nil
	}
	effectiveAt := r.EffectiveAt.UTC()
	return ScheduleApplication{
		RuleID:          r.ID,
		RuleEffectiveAt: effectiveAt,
		DueAt:           dueAt,
		Reason:          fmt.Sprintf("schedule rule %s effective %s", r.ID, effectiveAt.Format("2006-01-02")),
	}, true, nil
}

type DueWindow struct {
	DueAt        time.Time
	ReminderAt   time.Time
	OverdueAt    time.Time
	CurrentState InstrumentStatus
}

func BuildDueWindow(rule ScheduleRule, lastQualifiedAt, now time.Time) (DueWindow, error) {
	dueAt, err := rule.NextDue(lastQualifiedAt)
	if err != nil {
		return DueWindow{}, err
	}
	window := DueWindow{
		DueAt: dueAt, ReminderAt: dueAt.AddDate(0, 0, -rule.WarningDays),
		OverdueAt: dueAt, CurrentState: StatusQualified,
	}
	if !now.Before(window.OverdueAt) {
		window.CurrentState = StatusOverdue
	} else if !now.Before(window.ReminderAt) {
		window.CurrentState = StatusDueSoon
	}
	return window, nil
}
