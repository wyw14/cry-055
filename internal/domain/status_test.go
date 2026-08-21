package domain

import (
	"errors"
	"testing"
	"time"
)

func TestDerivedStatusPreservesBlockingState(t *testing.T) {
	now := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	for _, status := range []InstrumentStatus{StatusDisabled, StatusUnqualified, StatusReinspection} {
		if actual := DerivedStatus(status, now, now.Add(10*24*time.Hour), 30); actual != status {
			t.Fatalf("expected %s, got %s", status, actual)
		}
	}
}
func TestDerivedStatusByDueDate(t *testing.T) {
	now := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	cases := []struct {
		due  time.Time
		want InstrumentStatus
	}{{now.Add(-time.Minute), StatusOverdue}, {now.Add(5 * 24 * time.Hour), StatusDueSoon}, {now.Add(90 * 24 * time.Hour), StatusQualified}}
	for _, test := range cases {
		if got := DerivedStatus(StatusQualified, now, test.due, 30); got != test.want {
			t.Fatalf("due %s: want %s got %s", test.due, test.want, got)
		}
	}
}
func TestTransitionRejectsDirectRestore(t *testing.T) {
	if err := RequireTransition(StatusDisabled, StatusQualified); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}
}
