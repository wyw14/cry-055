package memory

import (
	"context"
	"sort"
	"strings"

	"github.com/wyw14/cry-055/internal/domain"
)

func (s *Store) CreateLaboratory(_ context.Context, laboratory domain.Laboratory) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.laboratories {
		if existing.Code == laboratory.Code {
			return domain.ErrDuplicate
		}
	}
	s.laboratories[laboratory.ID] = laboratory
	return nil
}

func (s *Store) GetLaboratory(_ context.Context, id domain.ID) (domain.Laboratory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.laboratories[id]
	if !ok {
		return domain.Laboratory{}, domain.ErrNotFound
	}
	return value, nil
}

func (s *Store) CreateInstrument(_ context.Context, instrument domain.Instrument) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.assetNumbers[instrument.AssetNumber]; exists {
		return domain.ErrDuplicate
	}
	if _, exists := s.instruments[instrument.ID]; exists {
		return domain.ErrDuplicate
	}
	s.instruments[instrument.ID] = instrument
	s.assetNumbers[instrument.AssetNumber] = instrument.ID
	return nil
}

func (s *Store) UpdateInstrument(_ context.Context, instrument domain.Instrument, expected domain.Version) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.instruments[instrument.ID]
	if !exists {
		return domain.ErrNotFound
	}
	if current.Version != expected {
		return domain.ErrConflict
	}
	if owner, exists := s.assetNumbers[instrument.AssetNumber]; exists && owner != instrument.ID {
		return domain.ErrDuplicate
	}
	s.instruments[instrument.ID] = instrument
	s.assetNumbers[instrument.AssetNumber] = instrument.ID
	return nil
}

func (s *Store) GetInstrument(_ context.Context, id domain.ID) (domain.Instrument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.instruments[id]
	if !ok {
		return domain.Instrument{}, domain.ErrNotFound
	}
	return value, nil
}

func (s *Store) GetInstrumentByAssetNumber(_ context.Context, assetNumber string) (domain.Instrument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.assetNumbers[strings.ToUpper(strings.TrimSpace(assetNumber))]
	if !ok {
		return domain.Instrument{}, domain.ErrNotFound
	}
	return s.instruments[id], nil
}

func (s *Store) ListInstruments(_ context.Context, request domain.PageRequest) (domain.Page[domain.Instrument], error) {
	s.mu.RLock()
	items := make([]domain.Instrument, 0, len(s.instruments))
	for _, item := range s.instruments {
		if !matchInstrument(item, request.Filters) {
			continue
		}
		items = append(items, item)
	}
	s.mu.RUnlock()
	sort.SliceStable(items, func(left, right int) bool {
		less := false
		switch request.Sort {
		case "asset_number":
			less = items[left].AssetNumber < items[right].AssetNumber
		case "next_due_at":
			less = items[left].NextDueAt.Before(items[right].NextDueAt)
		case "status":
			less = items[left].Status < items[right].Status
		default:
			less = items[left].CreatedAt.Before(items[right].CreatedAt)
		}
		if request.Desc {
			return !less
		}
		return less
	})
	return paginate(items, request.Page, request.Size), nil
}

func matchInstrument(item domain.Instrument, filters map[string]string) bool {
	for key, expected := range filters {
		switch key {
		case "laboratory_id":
			if string(item.LaboratoryID) != expected {
				return false
			}
		case "status":
			if string(item.Status) != expected {
				return false
			}
		case "criticality":
			if string(item.Criticality) != expected {
				return false
			}
		case "owner_id":
			if string(item.OwnerID) != expected {
				return false
			}
		}
	}
	return true
}

func paginate[T any](items []T, page, size int) domain.Page[T] {
	total := len(items)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}
	return domain.Page[T]{Items: append([]T(nil), items[start:end]...), Page: page, Size: size, Total: total}
}
