package domain

import (
	"strings"
	"time"
)

type PlanStatus string

const (
	PlanPending   PlanStatus = "pending"
	PlanInProcess PlanStatus = "in_process"
	PlanCompleted PlanStatus = "completed"
	PlanCancelled PlanStatus = "cancelled"
)

type CalibrationPlan struct {
	ID           ID         `json:"id"`
	InstrumentID ID         `json:"instrument_id"`
	ItemID       ID         `json:"item_id"`
	DueAt        time.Time  `json:"due_at"`
	Status       PlanStatus `json:"status"`
	AssignedTo   ID         `json:"assigned_to"`
	ChangeReason string     `json:"change_reason,omitempty"`
	Version      Version    `json:"version"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func NewPlan(instrumentID, itemID ID, dueAt time.Time, assignedTo ID, now time.Time) (CalibrationPlan, error) {
	if instrumentID.Empty() || itemID.Empty() {
		return CalibrationPlan{}, NewValidationError("plan", "instrument and item are required")
	}
	if dueAt.IsZero() {
		return CalibrationPlan{}, NewValidationError("due_at", "due date is required")
	}
	return CalibrationPlan{
		ID: NewID("plan"), InstrumentID: instrumentID, ItemID: itemID,
		DueAt: dueAt.UTC(), Status: PlanPending, AssignedTo: assignedTo,
		Version: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}, nil
}

func (p *CalibrationPlan) Reschedule(dueAt time.Time, reason string, expected Version, now time.Time) error {
	if p.Version != expected {
		return ErrConflict
	}
	if p.Status == PlanCompleted || p.Status == PlanCancelled {
		return ErrInvalidTransition
	}
	if dueAt.IsZero() || strings.TrimSpace(reason) == "" {
		return NewValidationError("change_reason", "new date and reason are required")
	}
	p.DueAt = dueAt.UTC()
	p.ChangeReason = strings.TrimSpace(reason)
	p.Version = p.Version.Next()
	p.UpdatedAt = now.UTC()
	return nil
}

func (p *CalibrationPlan) Complete(expected Version, now time.Time) error {
	if p.Version != expected {
		return ErrConflict
	}
	if p.Status == PlanCompleted || p.Status == PlanCancelled {
		return ErrInvalidTransition
	}
	p.Status = PlanCompleted
	p.Version = p.Version.Next()
	p.UpdatedAt = now.UTC()
	return nil
}
