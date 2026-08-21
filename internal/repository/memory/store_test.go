package memory

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

func TestConcurrentInstrumentUpdateAllowsOneVersion(t *testing.T) {
	store := New()
	ctx := context.Background()
	instrument := domain.Instrument{ID: "ins", AssetNumber: "A-1", Status: domain.StatusPending, Version: 1}
	if err := store.CreateInstrument(ctx, instrument); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var ready sync.WaitGroup
	ready.Add(2)
	var done sync.WaitGroup
	done.Add(2)
	var success atomic.Int32
	for index := 0; index < 2; index++ {
		go func(index int) {
			defer done.Done()
			candidate := instrument
			candidate.Name = string(rune('A' + index))
			candidate.Version = 2
			ready.Done()
			<-start
			if err := store.UpdateInstrument(ctx, candidate, 1); err == nil {
				success.Add(1)
			} else if !errors.Is(err, domain.ErrConflict) {
				t.Errorf("unexpected error: %v", err)
			}
		}(index)
	}
	ready.Wait()
	close(start)
	done.Wait()
	if success.Load() != 1 {
		t.Fatalf("expected one successful writer, got %d", success.Load())
	}
}

func TestTransactionRollbackRestoresAllAggregates(t *testing.T) {
	store := New()
	ctx := context.Background()
	instrument := domain.Instrument{ID: "ins", AssetNumber: "A-1", Status: domain.StatusPending, Version: 1}
	err := store.WithinTransaction(ctx, func(tx context.Context) error {
		if err := store.CreateInstrument(tx, instrument); err != nil {
			return err
		}
		return errors.New("forced rollback")
	})
	if err == nil {
		t.Fatal("expected rollback error")
	}
	if _, err := store.GetInstrument(ctx, "ins"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("instrument survived rollback: %v", err)
	}
}

func TestIdempotencyFingerprintConflict(t *testing.T) {
	store := New()
	ctx := context.Background()
	reserved, err := store.Reserve(ctx, "key", "first", time.Minute)
	if err != nil || !reserved {
		t.Fatalf("reserve failed: %v", err)
	}
	if _, err := store.Reserve(ctx, "key", "second", time.Minute); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}
