package domain

import "time"

type ScheduleRule struct {
	InstrumentID ID
	ItemID       ID
	PeriodDays   int
	WarningDays  int
	EffectiveAt  time.Time
	SupersedesID ID
}

func (r ScheduleRule) NextDue(lastQualifiedAt time.Time) (time.Time, error) {
	if r.InstrumentID.Empty() || r.ItemID.Empty() || r.PeriodDays <= 0 {
		return time.Time{}, NewValidationError("schedule_rule", "instrument, item and period are required")
	}
	anchor := lastQualifiedAt
	if anchor.Before(r.EffectiveAt) {
		anchor = r.EffectiveAt
	}
	return anchor.UTC().AddDate(0, 0, r.PeriodDays), nil
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
