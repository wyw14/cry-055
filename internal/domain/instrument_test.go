package domain

import (
	"errors"
	"testing"
	"time"
)

func TestCriticalOverdueInstrumentCannotBeUsed(t *testing.T) {
	now := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	instrument := Instrument{ID: "ins-1", Criticality: CriticalityCritical, Status: StatusQualified, NextDueAt: now.Add(-time.Second)}
	if instrument.CanBeUsed(now, 30) {
		t.Fatal("critical overdue instrument was allowed")
	}
}
func TestInstrumentStatusUsesOptimisticVersion(t *testing.T) {
	now := time.Now().UTC()
	instrument := Instrument{ID: "ins-1", Status: StatusPending, Version: 3}
	if err := instrument.ApplyStatus(StatusQualified, "", 2, now); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if instrument.Status != StatusPending {
		t.Fatal("conflicting update changed status")
	}
}
func TestBlockingStateRequiresReason(t *testing.T) {
	instrument := Instrument{ID: "ins-1", Status: StatusQualified, Version: 1}
	if err := instrument.ApplyStatus(StatusDisabled, "", 1, time.Now()); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
