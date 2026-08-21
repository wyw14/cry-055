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

type RestorationSnapshot struct {
	Actor                Actor
	InstrumentStatus     InstrumentStatus
	EffectiveStatus      *InstrumentStatus
	NonconformanceStatus NonconformanceStatus
	RetestID             ID
	CapturedAt           time.Time
}

type RestorationDecision struct {
	Actor        Actor
	SourceStatus InstrumentStatus
	TargetStatus InstrumentStatus
	RetestID     ID
}

func NewRestorationSnapshot(instrument Instrument, nonconformance Nonconformance, actor Actor, now time.Time) RestorationSnapshot {
	snapshot := RestorationSnapshot{
		Actor:                actor,
		InstrumentStatus:     instrument.Status,
		NonconformanceStatus: nonconformance.Status,
		RetestID:             nonconformance.RetestID,
		CapturedAt:           now.UTC(),
	}
	// The derived status is well-defined without a next-due date for blocking
	// states (disabled, unqualified, reinspection): DerivedStatus returns the
	// current status unchanged, so a fully reviewed historical instrument that
	// simply lacks scheduling info still has an effective status. Requiring a
	// due date here would let missing scheduling info negate an already
	// completed qualified retest, leaving the instrument stuck awaiting
	// restoration. The Authorize checks below still enforce the actor role,
	// restoration evidence, and status-order constraints independently.
	effective := DerivedStatus(instrument.Status, now, instrument.NextDueAt, 30)
	snapshot.EffectiveStatus = &effective
	return snapshot
}

func (s RestorationSnapshot) Authorize() (*RestorationDecision, error) {
	if s.Actor.ID.Empty() || !s.Actor.HasRole("quality_manager") {
		return nil, ErrUnauthorized
	}
	if s.NonconformanceStatus != NCRestorationReview || s.RetestID.Empty() {
		return nil, fmt.Errorf("%w: restoration evidence is incomplete", ErrInvalidTransition)
	}
	if s.EffectiveStatus == nil {
		return nil, fmt.Errorf("%w: effective status is unavailable", ErrInvalidTransition)
	}
	if s.InstrumentStatus != StatusReinspection || *s.EffectiveStatus != StatusReinspection {
		return nil, fmt.Errorf("%w: instrument is not awaiting restoration", ErrInvalidTransition)
	}
	if err := RequireTransition(s.InstrumentStatus, StatusQualified); err != nil {
		return nil, err
	}
	return &RestorationDecision{
		Actor:        s.Actor,
		SourceStatus: s.InstrumentStatus,
		TargetStatus: StatusQualified,
		RetestID:     s.RetestID,
	}, nil
}

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
