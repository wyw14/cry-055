package memory

import (
	"context"
	"sort"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

func (s *Store) AppendAudit(_ context.Context, value domain.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audits = append(s.audits, value)
	return nil
}

func (s *Store) ListAudit(_ context.Context, entityType string, entityID domain.ID) ([]domain.AuditEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.AuditEvent, 0)
	for _, value := range s.audits {
		if value.EntityType == entityType && value.EntityID == entityID {
			result = append(result, value)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return result, nil
}

func (s *Store) AppendUsageCheck(_ context.Context, value domain.UsageCheck) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usageChecks = append(s.usageChecks, value)
	return nil
}

func (s *Store) ListUsageChecks(_ context.Context, instrumentID domain.ID) ([]domain.UsageCheck, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.UsageCheck, 0)
	for _, value := range s.usageChecks {
		if value.InstrumentID == instrumentID {
			result = append(result, value)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CheckedAt.Before(result[j].CheckedAt) })
	return result, nil
}

func (s *Store) Reserve(_ context.Context, key, fingerprint string, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if current, exists := s.idempotency[key]; exists && current.ExpiresAt.After(now) {
		if current.Fingerprint != fingerprint {
			return false, domain.ErrConflict
		}
		return false, nil
	}
	s.idempotency[key] = idempotencyRecord{Fingerprint: fingerprint, ExpiresAt: now.Add(ttl)}
	return true, nil
}

func (s *Store) Complete(_ context.Context, key, fingerprint string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.idempotency[key]
	if !exists {
		return domain.ErrNotFound
	}
	if current.Fingerprint != fingerprint {
		return domain.ErrConflict
	}
	current.Completed = true
	s.idempotency[key] = current
	return nil
}

func (s *Store) Release(_ context.Context, key, fingerprint string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.idempotency[key]
	if !exists {
		return nil
	}
	if current.Fingerprint != fingerprint {
		return domain.ErrConflict
	}
	delete(s.idempotency, key)
	return nil
}
