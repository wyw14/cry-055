package memory

import (
	"context"
	"sync"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

type Store struct {
	mu             sync.RWMutex
	txMu           sync.Mutex
	laboratories   map[domain.ID]domain.Laboratory
	instruments    map[domain.ID]domain.Instrument
	assetNumbers   map[string]domain.ID
	items          map[domain.ID]domain.CalibrationItem
	plans          map[domain.ID]domain.CalibrationPlan
	executions     map[domain.ID]domain.CalibrationExecution
	executionRoots map[domain.ID][]domain.ID
	nonconformance map[domain.ID]domain.Nonconformance
	certificates   map[domain.ID]domain.Certificate
	alerts         map[domain.ID]domain.Alert
	alertDedup     map[string]domain.ID
	audits         []domain.AuditEvent
	usageChecks    []domain.UsageCheck
	idempotency    map[string]idempotencyRecord
}

type idempotencyRecord struct {
	Fingerprint string
	Completed   bool
	ExpiresAt   time.Time
}

func New() *Store {
	return &Store{
		laboratories: make(map[domain.ID]domain.Laboratory), instruments: make(map[domain.ID]domain.Instrument),
		assetNumbers: make(map[string]domain.ID), items: make(map[domain.ID]domain.CalibrationItem),
		plans: make(map[domain.ID]domain.CalibrationPlan), executions: make(map[domain.ID]domain.CalibrationExecution),
		executionRoots: make(map[domain.ID][]domain.ID), nonconformance: make(map[domain.ID]domain.Nonconformance),
		certificates: make(map[domain.ID]domain.Certificate), alerts: make(map[domain.ID]domain.Alert),
		alertDedup: make(map[string]domain.ID), idempotency: make(map[string]idempotencyRecord),
	}
}

func (s *Store) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	s.txMu.Lock()
	defer s.txMu.Unlock()
	snapshot := s.snapshot()
	if err := fn(ctx); err != nil {
		s.restore(snapshot)
		return err
	}
	return nil
}

type snapshot struct {
	laboratories   map[domain.ID]domain.Laboratory
	instruments    map[domain.ID]domain.Instrument
	assetNumbers   map[string]domain.ID
	items          map[domain.ID]domain.CalibrationItem
	plans          map[domain.ID]domain.CalibrationPlan
	executions     map[domain.ID]domain.CalibrationExecution
	executionRoots map[domain.ID][]domain.ID
	nonconformance map[domain.ID]domain.Nonconformance
	certificates   map[domain.ID]domain.Certificate
	alerts         map[domain.ID]domain.Alert
	alertDedup     map[string]domain.ID
	audits         []domain.AuditEvent
	usageChecks    []domain.UsageCheck
	idempotency    map[string]idempotencyRecord
}

func (s *Store) snapshot() snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return snapshot{
		laboratories: cloneMap(s.laboratories), instruments: cloneMap(s.instruments), assetNumbers: cloneMap(s.assetNumbers),
		items: cloneMap(s.items), plans: cloneMap(s.plans), executions: cloneMap(s.executions),
		executionRoots: cloneSliceMap(s.executionRoots), nonconformance: cloneMap(s.nonconformance),
		certificates: cloneMap(s.certificates), alerts: cloneMap(s.alerts), alertDedup: cloneMap(s.alertDedup),
		audits: append([]domain.AuditEvent(nil), s.audits...), usageChecks: append([]domain.UsageCheck(nil), s.usageChecks...),
		idempotency: cloneMap(s.idempotency),
	}
}

func (s *Store) restore(value snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.laboratories, s.instruments, s.assetNumbers = value.laboratories, value.instruments, value.assetNumbers
	s.items, s.plans, s.executions = value.items, value.plans, value.executions
	s.executionRoots, s.nonconformance, s.certificates = value.executionRoots, value.nonconformance, value.certificates
	s.alerts, s.alertDedup, s.audits = value.alerts, value.alertDedup, value.audits
	s.usageChecks, s.idempotency = value.usageChecks, value.idempotency
}

func cloneMap[K comparable, V any](source map[K]V) map[K]V {
	target := make(map[K]V, len(source))
	for key, value := range source {
		target[key] = value
	}
	return target
}

func cloneSliceMap[K comparable, V any](source map[K][]V) map[K][]V {
	target := make(map[K][]V, len(source))
	for key, value := range source {
		target[key] = append([]V(nil), value...)
	}
	return target
}
