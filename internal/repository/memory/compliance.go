package memory

import (
	"context"
	"sort"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

func (s *Store) CreateNonconformance(_ context.Context, value domain.Nonconformance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.nonconformance {
		if existing.InstrumentID == value.InstrumentID && existing.Status != domain.NCClosed {
			return domain.ErrDuplicate
		}
	}
	s.nonconformance[value.ID] = value
	return nil
}

func (s *Store) UpdateNonconformance(_ context.Context, value domain.Nonconformance, expected domain.Version) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.nonconformance[value.ID]
	if !ok {
		return domain.ErrNotFound
	}
	if current.Version != expected {
		return domain.ErrConflict
	}
	s.nonconformance[value.ID] = value
	return nil
}

func (s *Store) GetNonconformance(_ context.Context, id domain.ID) (domain.Nonconformance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.nonconformance[id]
	if !ok {
		return domain.Nonconformance{}, domain.ErrNotFound
	}
	return value, nil
}

func (s *Store) FindOpenByInstrument(_ context.Context, instrumentID domain.ID) (domain.Nonconformance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, value := range s.nonconformance {
		if value.InstrumentID == instrumentID && value.Status != domain.NCClosed {
			return value, nil
		}
	}
	return domain.Nonconformance{}, domain.ErrNotFound
}

func (s *Store) CreateCertificate(_ context.Context, value domain.Certificate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, current := range s.certificates {
		if current.Number == value.Number {
			return domain.ErrDuplicate
		}
	}
	s.certificates[value.ID] = value
	return nil
}

func (s *Store) GetCertificate(_ context.Context, id domain.ID) (domain.Certificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.certificates[id]
	if !ok {
		return domain.Certificate{}, domain.ErrNotFound
	}
	return value, nil
}

func (s *Store) ListExpiringCertificates(_ context.Context, from, to time.Time) ([]domain.Certificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Certificate, 0)
	for _, value := range s.certificates {
		if !value.ExpiresAt.Before(from) && !value.ExpiresAt.After(to) {
			result = append(result, value)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ExpiresAt.Before(result[j].ExpiresAt) })
	return result, nil
}

func (s *Store) UpsertAlert(_ context.Context, value domain.Alert) (domain.Alert, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, exists := s.alertDedup[value.Deduplication]; exists {
		return s.alerts[id], false, nil
	}
	s.alerts[value.ID] = value
	s.alertDedup[value.Deduplication] = value.ID
	return value, true, nil
}

func (s *Store) AcknowledgeAlert(_ context.Context, value domain.Alert) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.alerts[value.ID]; !ok {
		return domain.ErrNotFound
	}
	s.alerts[value.ID] = value
	return nil
}

func (s *Store) ListOpenAlerts(_ context.Context, request domain.PageRequest) (domain.Page[domain.Alert], error) {
	s.mu.RLock()
	items := make([]domain.Alert, 0)
	for _, value := range s.alerts {
		if value.AcknowledgedAt != nil {
			continue
		}
		if severity := request.Filters["severity"]; severity != "" && string(value.Severity) != severity {
			continue
		}
		if kind := request.Filters["kind"]; kind != "" && value.Kind != kind {
			continue
		}
		if id := request.Filters["instrument_id"]; id != "" && string(value.InstrumentID) != id {
			continue
		}
		items = append(items, value)
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return paginate(items, request.Page, request.Size), nil
}
