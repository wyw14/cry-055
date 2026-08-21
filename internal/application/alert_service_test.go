package application

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
	"github.com/wyw14/cry-055/internal/platform/localnotify"
)

func TestAlertScanDeduplicatesNotification(t *testing.T) {
	fixture := newFixture(t)
	instrument := fixture.instrument
	instrument.Status = domain.StatusQualified
	instrument.NextDueAt = fixture.clock.Now().Add(2 * time.Hour)
	instrument.Version = 2
	if err := fixture.store.UpdateInstrument(context.Background(), instrument, 1); err != nil {
		t.Fatal(err)
	}
	notifier := localnotify.New()
	service := NewAlertService(fixture.store, fixture.store, fixture.store, notifier, fixture.clock)
	for index := 0; index < 2; index++ {
		_, _, err := service.ScanInstrument(context.Background(), instrument.ID, 30)
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := len(notifier.Sent()); got != 1 {
		t.Fatalf("expected one notification, got %d", got)
	}
}
