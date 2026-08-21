package application

import (
	"context"
	"errors"
	"testing"

	"github.com/wyw14/cry-055/internal/domain"
	"github.com/wyw14/cry-055/internal/repository/memory"
)

type failingReviewAudit struct {
	*memory.Store
	err error
}

func (s failingReviewAudit) AppendAudit(context.Context, domain.AuditEvent) error { return s.err }

func TestReviewAuditFailurePreservesCauseAndRollsBackDecision(t *testing.T) {
	fixture := newFixture(t)
	measurement, err := domain.NewMeasurement("span", 100, 100.1, fixture.item.Tolerance)
	if err != nil {
		t.Fatal(err)
	}
	execution, err := domain.NewExecution(fixture.instrument.ID, fixture.item.ID, fixture.plan.ID, "operator", []domain.Measurement{measurement}, "certificate-1", fixture.clock.Now(), fixture.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.store.CreateExecution(context.Background(), execution); err != nil {
		t.Fatal(err)
	}
	auditFailure := errors.New("audit sink unavailable")
	audits := failingReviewAudit{Store: fixture.store, err: auditFailure}
	service := NewExecutionService(fixture.store, fixture.store, fixture.store, fixture.store, audits, fixture.clock)
	ctx := WithRequestMetadata(context.Background(), RequestMetadata{RequestID: "request-review-1", Actor: domain.Actor{ID: "reviewer-1"}})

	_, err = service.Review(ctx, execution.ID, "reviewer-1", "measurement evidence accepted")
	if !errors.Is(err, auditFailure) {
		t.Errorf("review error lost audit cause: %v", err)
	}
	stored, err := fixture.store.GetExecution(context.Background(), execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ReviewedAt != nil || !stored.ReviewerID.Empty() || stored.Version != execution.Version {
		t.Fatalf("failed review left persisted decision: reviewer=%s reviewed_at=%v version=%d", stored.ReviewerID, stored.ReviewedAt, stored.Version)
	}
	events, err := fixture.store.ListAudit(context.Background(), "calibration_execution", execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("failed review left %d audit events", len(events))
	}
}
