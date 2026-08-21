package application

import (
	"context"
	"errors"
	"testing"

	"github.com/wyw14/cry-055/internal/domain"
	"github.com/wyw14/cry-055/internal/repository/memory"
)

type failingRevisionRepository struct {
	*memory.Store
	failure  error
	stagedID domain.ID
}

func (r *failingRevisionRepository) StageExecutionRevision(ctx context.Context, execution domain.CalibrationExecution) error {
	r.stagedID = execution.ID
	return r.Store.StageExecutionRevision(ctx, execution)
}

func (r *failingRevisionRepository) LinkExecutionRevision(context.Context, domain.CalibrationExecution) error {
	return r.failure
}

func TestRevisionFailurePreservesImmutableVersionChain(t *testing.T) {
	fixture := newFixture(t)
	ctx := context.Background()
	originalMeasurement, err := domain.NewMeasurement("span", 100, 100.2, fixture.item.Tolerance)
	if err != nil {
		t.Fatal(err)
	}
	original, err := domain.NewExecution(fixture.instrument.ID, fixture.item.ID, fixture.plan.ID, "operator", []domain.Measurement{originalMeasurement}, "", fixture.clock.Now(), fixture.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.store.CreateExecution(ctx, original); err != nil {
		t.Fatal(err)
	}

	service := NewExecutionService(fixture.store, fixture.store, fixture.store, fixture.store, fixture.clock)
	correctedMeasurement, _ := domain.NewMeasurement("span", 100, 100.1, fixture.item.Tolerance)
	firstRevision, err := service.Revise(ctx, original.ID, "corrector-1", []domain.Measurement{correctedMeasurement})
	if err != nil {
		t.Fatal(err)
	}
	if firstRevision.RootID != original.ID || firstRevision.PreviousID != original.ID || firstRevision.Version != 2 {
		t.Errorf("first revision broke chain metadata: root=%s previous=%s version=%d", firstRevision.RootID, firstRevision.PreviousID, firstRevision.Version)
	}

	linkFailure := errors.New("revision head unavailable")
	failingRepository := &failingRevisionRepository{Store: fixture.store, failure: linkFailure}
	failingService := NewExecutionService(failingRepository, fixture.store, fixture.store, fixture.store, fixture.clock)
	failedMeasurement, _ := domain.NewMeasurement("span", 100, 99.9, fixture.item.Tolerance)
	_, err = failingService.Revise(ctx, firstRevision.ID, "corrector-2", []domain.Measurement{failedMeasurement})
	if !errors.Is(err, linkFailure) {
		t.Errorf("expected link failure in error chain, got %v", err)
	}
	if failingRepository.stagedID.Empty() {
		t.Fatal("revision was not staged before the injected link failure")
	}
	if _, lookupErr := fixture.store.GetExecution(ctx, failingRepository.stagedID); !errors.Is(lookupErr, domain.ErrNotFound) {
		t.Errorf("failed revision survived rollback: %v", lookupErr)
	}

	retryMeasurement, _ := domain.NewMeasurement("span", 100, 100.05, fixture.item.Tolerance)
	if _, err := service.Revise(ctx, firstRevision.ID, "corrector-3", []domain.Measurement{retryMeasurement}); err != nil {
		t.Fatalf("retry after rollback failed: %v", err)
	}
	versions, err := fixture.store.ListExecutionVersions(ctx, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 3 {
		t.Fatalf("expected three committed versions, got %d", len(versions))
	}
	for index, execution := range versions {
		expectedVersion := domain.Version(index + 1)
		if execution.RootID != original.ID || execution.Version != expectedVersion {
			t.Errorf("version %d has root=%s number=%d", index, execution.RootID, execution.Version)
		}
		if index > 0 && execution.PreviousID != versions[index-1].ID {
			t.Errorf("version %d points to %s instead of %s", index, execution.PreviousID, versions[index-1].ID)
		}
	}
	storedOriginal, err := fixture.store.GetExecution(ctx, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedOriginal.Measurements[0] != originalMeasurement {
		t.Error("original calibration result was mutated")
	}
}
