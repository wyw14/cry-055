package application

import (
	"context"
	"errors"
	"testing"

	"github.com/wyw14/cry-055/internal/domain"
)

func TestUnqualifiedExecutionBlocksInstrumentAndOpensCase(t *testing.T) {
	fixture := newFixture(t)
	service := NewExecutionService(fixture.store, fixture.store, fixture.store, fixture.store, fixture.clock)
	measurement, _ := domain.NewMeasurement("span", 100, 101, fixture.item.Tolerance)
	execution, err := service.Record(context.Background(), ExecutionInput{InstrumentID: fixture.instrument.ID, ItemID: fixture.item.ID, PlanID: fixture.plan.ID, ExecutorID: "operator", Measurements: []domain.Measurement{measurement}, CompletedAt: fixture.clock.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if execution.Conclusion != domain.ConclusionUnqualified {
		t.Fatal("expected unqualified conclusion")
	}
	instrument, err := fixture.store.GetInstrument(context.Background(), fixture.instrument.ID)
	if err != nil {
		t.Fatal(err)
	}
	if instrument.Status != domain.StatusUnqualified {
		t.Fatalf("expected instrument blocked, got %s", instrument.Status)
	}
	if _, err := fixture.store.FindOpenByInstrument(context.Background(), instrument.ID); err != nil {
		t.Fatalf("missing nonconformance: %v", err)
	}
}

func TestExecutionTransactionRollsBackOnDuplicateCase(t *testing.T) {
	fixture := newFixture(t)
	existing, _ := domain.NewNonconformance(fixture.instrument.ID, "old-exec", "existing", fixture.clock.Now())
	if err := fixture.store.CreateNonconformance(context.Background(), existing); err != nil {
		t.Fatal(err)
	}
	service := NewExecutionService(fixture.store, fixture.store, fixture.store, fixture.store, fixture.clock)
	measurement, _ := domain.NewMeasurement("span", 100, 101, fixture.item.Tolerance)
	_, err := service.Record(context.Background(), ExecutionInput{InstrumentID: fixture.instrument.ID, ItemID: fixture.item.ID, PlanID: fixture.plan.ID, ExecutorID: "operator", Measurements: []domain.Measurement{measurement}, CompletedAt: fixture.clock.Now()})
	if !errors.Is(err, domain.ErrDuplicate) {
		t.Fatalf("expected duplicate case, got %v", err)
	}
	instrument, _ := fixture.store.GetInstrument(context.Background(), fixture.instrument.ID)
	if instrument.Status != domain.StatusPending {
		t.Fatalf("partial instrument update survived rollback: %s", instrument.Status)
	}
	plan, _ := fixture.store.GetPlan(context.Background(), fixture.plan.ID)
	if plan.Status != domain.PlanPending {
		t.Fatalf("partial plan update survived rollback: %s", plan.Status)
	}
}
