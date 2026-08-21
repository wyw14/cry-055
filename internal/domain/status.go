package domain

import (
	"fmt"
	"time"
)

type InstrumentStatus string

const (
	StatusPending      InstrumentStatus = "pending"
	StatusQualified    InstrumentStatus = "qualified"
	StatusDueSoon      InstrumentStatus = "due_soon"
	StatusOverdue      InstrumentStatus = "overdue"
	StatusUnqualified  InstrumentStatus = "unqualified"
	StatusDisabled     InstrumentStatus = "disabled"
	StatusReinspection InstrumentStatus = "reinspection"
)

func (s InstrumentStatus) Valid() bool {
	switch s {
	case StatusPending, StatusQualified, StatusDueSoon, StatusOverdue, StatusUnqualified, StatusDisabled, StatusReinspection:
		return true
	default:
		return false
	}
}

func (s InstrumentStatus) AllowsUse(critical bool) bool {
	if critical {
		return s == StatusQualified || s == StatusDueSoon
	}
	return s == StatusQualified || s == StatusDueSoon || s == StatusPending
}

func DerivedStatus(current InstrumentStatus, now, dueAt time.Time, warningDays int) InstrumentStatus {
	if current == StatusDisabled || current == StatusUnqualified || current == StatusReinspection {
		return current
	}
	if !dueAt.After(now) {
		return StatusOverdue
	}
	if dueAt.Sub(now) <= time.Duration(warningDays)*24*time.Hour {
		return StatusDueSoon
	}
	return StatusQualified
}

func RequireTransition(from, to InstrumentStatus) error {
	allowed := map[InstrumentStatus]map[InstrumentStatus]bool{
		StatusPending:      {StatusQualified: true, StatusUnqualified: true, StatusDisabled: true},
		StatusQualified:    {StatusDueSoon: true, StatusOverdue: true, StatusUnqualified: true, StatusDisabled: true},
		StatusDueSoon:      {StatusQualified: true, StatusOverdue: true, StatusUnqualified: true, StatusDisabled: true},
		StatusOverdue:      {StatusReinspection: true, StatusDisabled: true},
		StatusUnqualified:  {StatusReinspection: true, StatusDisabled: true},
		StatusDisabled:     {StatusReinspection: true},
		StatusReinspection: {StatusQualified: true, StatusUnqualified: true, StatusDisabled: true},
	}
	if !to.Valid() || !allowed[from][to] {
		return fmt.Errorf("%w: %s to %s", ErrInvalidTransition, from, to)
	}
	return nil
}
