package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNonconformanceRequiresCompleteRecoveryWorkflow(t *testing.T) {
	now := time.Now().UTC()
	nc, err := NewNonconformance("ins", "exec", "outside tolerance", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := nc.AssessImpact([]ImpactedBatch{{BatchNumber: "B-1", UsedAt: now, Product: "sample"}}, 1, now); err != nil {
		t.Fatal(err)
	}
	if err := nc.RequestRetest(2, now); err != nil {
		t.Fatal(err)
	}
	if err := nc.AttachRetest("exec-2", true, 3, now); err != nil {
		t.Fatal(err)
	}
	actor := Actor{ID: "quality", Roles: []string{"quality_manager"}}
	if err := nc.ConfirmRestoration(actor, "evidence reviewed", 4, now); err != nil {
		t.Fatal(err)
	}
	if nc.Status != NCClosed {
		t.Fatalf("expected closed, got %s", nc.Status)
	}
}
func TestNonconformanceRejectsDuplicateBatch(t *testing.T) {
	now := time.Now().UTC()
	nc, _ := NewNonconformance("ins", "exec", "reason", now)
	err := nc.AssessImpact([]ImpactedBatch{{BatchNumber: "B-1", UsedAt: now}, {BatchNumber: "B-1", UsedAt: now}}, 1, now)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
