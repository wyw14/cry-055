package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

func TestUsageCheckPersistsBlockedAttempt(t *testing.T) {
	fixture := newFixture(t)
	instrument := fixture.instrument
	instrument.Status = domain.StatusQualified
	instrument.NextDueAt = fixture.clock.Now().Add(-time.Hour)
	instrument.Version = 2
	if err := fixture.store.UpdateInstrument(context.Background(), instrument, 1); err != nil {
		t.Fatal(err)
	}
	service := NewUsageService(fixture.store, fixture.store, fixture.clock, 30)
	check, err := service.Check(context.Background(), instrument.ID, "BATCH-9", "operator")
	if !errors.Is(err, domain.ErrInstrumentBlocked) {
		t.Fatalf("expected blocked error, got %v", err)
	}
	if check.Decision != domain.UsageBlocked {
		t.Fatalf("expected blocked decision, got %s", check.Decision)
	}
	history, err := service.History(context.Background(), instrument.ID)
	if err != nil || len(history) != 1 {
		t.Fatalf("blocked attempt not persisted: %v %#v", err, history)
	}
}
