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

// Restoration must be rejected while the qualified retest is still pending
// (awaiting_retest): a retest has only been requested, not attached and
// persisted. Authorizing here would let a concurrent restore close the case
// before the retest evidence lands, and the attach then loses to a version
// conflict, silently dropping the retest.
func TestNonconformanceRejectsRestorationBeforeRetestAttached(t *testing.T) {
	now := time.Now().UTC()
	nc, _ := NewNonconformance("ins", "exec", "outside tolerance", now)
	_ = nc.AssessImpact([]ImpactedBatch{{BatchNumber: "B-1", UsedAt: now, Product: "sample"}}, 1, now)
	_ = nc.RequestRetest(2, now)
	// state is now awaiting_retest, version 3, no qualified retest attached.

	manager := Actor{ID: "quality", Roles: []string{"quality_manager"}}

	// Authorization is rejected: retest requested, but no reviewed qualified
	// retest evidence has landed in the store yet.
	if _, err := nc.CollectRestorationEvidence().Authorize(manager, "evidence reviewed", 3); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected restoration rejected before retest attached, got %v", err)
	}

	// Even if a caller bypassed Authorize, ApplyRestoration must refuse to
	// close from awaiting_retest and must not advance the version.
	auth := RestorationAuthorization{CaseVersion: 3, ActorID: manager.ID, RetestID: "", Comment: "reviewed"}
	if err := nc.ApplyRestoration(auth, now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected apply restoration rejected, got %v", err)
	}
	if nc.Status != NCAwaitingRetest || nc.Version != 3 {
		t.Fatalf("case must remain awaiting_retest at v3, got status=%s v=%d", nc.Status, nc.Version)
	}
}
