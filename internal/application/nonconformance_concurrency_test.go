package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

type caseActorKey struct{}

const (
	attachActor  = "attach"
	restoreActor = "restore"
)

type sharedCaseGate struct {
	NonconformanceRepository

	mu              sync.Mutex
	readers         int
	bothRead        chan struct{}
	restoreFinished chan struct{}
	readyOnce       sync.Once
}

func (g *sharedCaseGate) GetNonconformance(ctx context.Context, id domain.ID) (domain.Nonconformance, error) {
	value, err := g.NonconformanceRepository.GetNonconformance(ctx, id)
	if err != nil {
		return domain.Nonconformance{}, err
	}
	g.mu.Lock()
	g.readers++
	if g.readers == 2 {
		g.readyOnce.Do(func() { close(g.bothRead) })
	}
	g.mu.Unlock()
	<-g.bothRead
	if actor, _ := ctx.Value(caseActorKey{}).(string); actor == attachActor {
		<-g.restoreFinished
	}
	return value, nil
}

func TestConcurrentRetestAndRestorationRequireQualifiedEvidence(t *testing.T) {
	fx := newFixture(t)
	ctx := context.Background()

	nc, err := domain.NewNonconformance(fx.instrument.ID, "exec-failed", "outside tolerance", fx.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	batches := []domain.ImpactedBatch{{BatchNumber: "B-CON-1", UsedAt: fx.clock.Now().Add(-time.Hour), Product: "sterility sample", Disposition: "quarantined"}}
	if err := nc.AssessImpact(batches, nc.Version, fx.clock.Now()); err != nil {
		t.Fatal(err)
	}
	if err := nc.RequestRetest(nc.Version, fx.clock.Now()); err != nil {
		t.Fatal(err)
	}
	if err := fx.store.CreateNonconformance(ctx, nc); err != nil {
		t.Fatal(err)
	}

	measurement, err := domain.NewMeasurement("100 kPa", 100, 100.1, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	retest, err := domain.NewExecution(fx.instrument.ID, fx.item.ID, fx.plan.ID, "retest-operator", []domain.Measurement{measurement}, "cert-retest", fx.clock.Now(), fx.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := retest.Review("independent-reviewer", "qualified result confirmed", fx.clock.Now()); err != nil {
		t.Fatal(err)
	}
	if err := fx.store.CreateExecution(ctx, retest); err != nil {
		t.Fatal(err)
	}

	gate := &sharedCaseGate{
		NonconformanceRepository: fx.store,
		bothRead:                 make(chan struct{}),
		restoreFinished:          make(chan struct{}),
	}
	service := NewNonconformanceService(gate, fx.store, fx.store, fx.store, fx.clock)
	start := make(chan struct{})
	type outcome struct {
		actor string
		err   error
	}
	outcomes := make(chan outcome, 2)

	go func() {
		<-start
		actorCtx := context.WithValue(ctx, caseActorKey{}, attachActor)
		_, callErr := service.AttachRetest(actorCtx, nc.ID, retest.ID, nc.Version)
		outcomes <- outcome{actor: attachActor, err: callErr}
	}()
	go func() {
		<-start
		actorCtx := context.WithValue(ctx, caseActorKey{}, restoreActor)
		manager := domain.Actor{ID: "quality-manager", Roles: []string{"quality_manager"}}
		_, callErr := service.Restore(actorCtx, nc.ID, manager, "retest evidence reviewed", nc.Version)
		close(gate.restoreFinished)
		outcomes <- outcome{actor: restoreActor, err: callErr}
	}()
	close(start)

	results := map[string]error{}
	for range 2 {
		result := <-outcomes
		results[result.actor] = result.err
	}
	if !errors.Is(results[restoreActor], domain.ErrUnauthorized) {
		t.Fatalf("expected early restoration to be rejected, got %v", results[restoreActor])
	}
	if results[attachActor] != nil {
		t.Fatalf("expected reviewed qualified retest to attach, got %v", results[attachActor])
	}

	stored, err := fx.store.GetNonconformance(ctx, nc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != domain.NCRestorationReview || stored.RetestID != retest.ID || !stored.RestoreActorID.Empty() {
		t.Fatalf("expected restoration review with attached retest and no restorer, got status=%s retest=%s restorer=%s", stored.Status, stored.RetestID, stored.RestoreActorID)
	}
}
