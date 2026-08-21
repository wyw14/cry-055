package application

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

func TestRestoreAuthorizesQualityManagerWithPartialInstrumentState(t *testing.T) {
	fx := newFixture(t)
	ctx := context.Background()
	instrument := fx.instrument
	transitions := []struct {
		status domain.InstrumentStatus
		reason string
	}{
		{domain.StatusQualified, ""},
		{domain.StatusDueSoon, ""},
		{domain.StatusOverdue, ""},
		{domain.StatusReinspection, ""},
		{domain.StatusUnqualified, "retest outside tolerance"},
		{domain.StatusDisabled, "quality hold"},
		{domain.StatusReinspection, ""},
	}
	for _, transition := range transitions {
		if err := instrument.ApplyStatus(transition.status, transition.reason, instrument.Version, fx.clock.Now()); err != nil {
			t.Fatalf("transition to %s: %v", transition.status, err)
		}
	}
	if !instrument.NextDueAt.IsZero() {
		t.Fatalf("expected partial instrument state without a due date")
	}
	if err := fx.store.UpdateInstrument(ctx, instrument, fx.instrument.Version); err != nil {
		t.Fatal(err)
	}

	nc, err := domain.NewNonconformance(instrument.ID, "exec-failed", "outside tolerance", fx.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	batches := []domain.ImpactedBatch{{BatchNumber: "B-REC-1", UsedAt: fx.clock.Now().Add(-time.Hour), Product: "reference material", Disposition: "released"}}
	if err := nc.AssessImpact(batches, nc.Version, fx.clock.Now()); err != nil {
		t.Fatal(err)
	}
	if err := nc.RequestRetest(nc.Version, fx.clock.Now()); err != nil {
		t.Fatal(err)
	}
	if err := nc.AttachRetest("exec-qualified", true, nc.Version, fx.clock.Now()); err != nil {
		t.Fatal(err)
	}
	if err := fx.store.CreateNonconformance(ctx, nc); err != nil {
		t.Fatal(err)
	}

	manager := domain.Actor{ID: "quality-manager", Roles: []string{"quality_manager"}}
	ctx = WithRequestMetadata(ctx, RequestMetadata{RequestID: "restore-req", Actor: manager})
	service := NewNonconformanceService(fx.store, fx.store, fx.store, fx.store, fx.clock)
	restored, err := service.Restore(ctx, nc.ID, manager, "qualified retest reviewed", nc.Version)
	if err != nil {
		t.Fatalf("restore fully reviewed case: %v", err)
	}
	if restored.Status != domain.NCClosed || restored.RestoreActorID != manager.ID {
		t.Fatalf("expected closed case restored by %s, got status=%s actor=%s", manager.ID, restored.Status, restored.RestoreActorID)
	}
	storedInstrument, err := fx.store.GetInstrument(ctx, instrument.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedInstrument.Status != domain.StatusQualified || storedInstrument.DisabledReason != "" {
		t.Fatalf("expected qualified instrument with cleared hold, got status=%s reason=%q", storedInstrument.Status, storedInstrument.DisabledReason)
	}
}
