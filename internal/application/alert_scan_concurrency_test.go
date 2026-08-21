package application

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
	"github.com/wyw14/cry-055/internal/platform/localnotify"
	"github.com/wyw14/cry-055/internal/repository/memory"
)

type alertScanGate struct {
	mu      sync.Mutex
	arrived int
	release chan struct{}
}

func newAlertScanGate() *alertScanGate {
	return &alertScanGate{release: make(chan struct{})}
}

func (g *alertScanGate) waitForBoth() {
	g.mu.Lock()
	g.arrived++
	if g.arrived == 2 {
		close(g.release)
	}
	g.mu.Unlock()
	<-g.release
}

type gatedAlertRepository struct {
	store *memory.Store
	gate  *alertScanGate
}

func (r *gatedAlertRepository) FindAlertByDeduplication(ctx context.Context, key string) (domain.Alert, error) {
	alert, err := r.store.FindAlertByDeduplication(ctx, key)
	r.gate.waitForBoth()
	return alert, err
}

func (r *gatedAlertRepository) RecordAlertScan(ctx context.Context, alert domain.Alert, audit domain.AuditEvent) error {
	return r.store.RecordAlertScan(ctx, alert, audit)
}

func (r *gatedAlertRepository) CommitAlertScan(ctx context.Context, alert domain.Alert, audit domain.AuditEvent) (domain.Alert, bool, error) {
	r.gate.waitForBoth()
	return r.store.CommitAlertScan(ctx, alert, audit)
}

func (r *gatedAlertRepository) AcknowledgeAlert(ctx context.Context, alert domain.Alert) error {
	return r.store.AcknowledgeAlert(ctx, alert)
}

func (r *gatedAlertRepository) ListOpenAlerts(ctx context.Context, page domain.PageRequest) (domain.Page[domain.Alert], error) {
	return r.store.ListOpenAlerts(ctx, page)
}

func TestConcurrentAlertScanCommitsNotificationAndAuditOnce(t *testing.T) {
	fixture := newFixture(t)
	instrument := fixture.instrument
	instrument.Status = domain.StatusQualified
	instrument.NextDueAt = fixture.clock.Now().Add(2 * time.Hour)
	instrument.Version = 2
	if err := fixture.store.UpdateInstrument(context.Background(), instrument, 1); err != nil {
		t.Fatal(err)
	}

	notifier := localnotify.New()
	alerts := &gatedAlertRepository{store: fixture.store, gate: newAlertScanGate()}
	service := NewAlertService(alerts, fixture.store, fixture.store, notifier, fixture.clock)

	start := make(chan struct{})
	var ready sync.WaitGroup
	var done sync.WaitGroup
	var created atomic.Int32
	errorsByActor := make(chan error, 2)
	ready.Add(2)
	done.Add(2)
	for actor := 0; actor < 2; actor++ {
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			_, wasCreated, err := service.ScanInstrument(context.Background(), instrument.ID, 30)
			if wasCreated {
				created.Add(1)
			}
			errorsByActor <- err
		}()
	}
	ready.Wait()
	close(start)
	done.Wait()
	close(errorsByActor)

	for err := range errorsByActor {
		if err != nil {
			t.Fatalf("scan failed: %v", err)
		}
	}
	if got := created.Load(); got != 1 {
		t.Errorf("expected one scan to create the alert, got %d", got)
	}
	if got := len(notifier.Sent()); got != 1 {
		t.Errorf("expected one notification, got %d", got)
	}
	audits, err := fixture.store.ListAudit(context.Background(), "instrument", instrument.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(audits); got != 1 {
		t.Errorf("expected one alert audit event, got %d", got)
	}
	page, err := fixture.store.ListOpenAlerts(context.Background(), domain.PageRequest{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Errorf("expected one open alert, total=%d items=%d", page.Total, len(page.Items))
	}
}
