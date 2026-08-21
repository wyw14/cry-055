package memory

import (
	"context"
	"sort"
	"strings"

	"github.com/wyw14/cry-055/internal/domain"
)

func (s *Store) CreateItem(_ context.Context, item domain.CalibrationItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.items {
		if existing.Code == item.Code {
			return domain.ErrDuplicate
		}
	}
	s.items[item.ID] = item
	return nil
}

func (s *Store) GetItem(_ context.Context, id domain.ID) (domain.CalibrationItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[id]
	if !ok {
		return domain.CalibrationItem{}, domain.ErrNotFound
	}
	return item, nil
}

func (s *Store) GetItemByCode(_ context.Context, code string) (domain.CalibrationItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	code = strings.ToUpper(strings.TrimSpace(code))
	for _, item := range s.items {
		if item.Code == code {
			return item, nil
		}
	}
	return domain.CalibrationItem{}, domain.ErrNotFound
}

func (s *Store) CreatePlan(_ context.Context, plan domain.CalibrationPlan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.plans {
		if existing.InstrumentID == plan.InstrumentID && existing.ItemID == plan.ItemID && existing.Status != domain.PlanCompleted && existing.Status != domain.PlanCancelled {
			return domain.ErrDuplicate
		}
	}
	s.plans[plan.ID] = plan
	return nil
}

func (s *Store) UpdatePlan(_ context.Context, plan domain.CalibrationPlan, expected domain.Version) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.plans[plan.ID]
	if !ok {
		return domain.ErrNotFound
	}
	if current.Version != expected {
		return domain.ErrConflict
	}
	s.plans[plan.ID] = plan
	return nil
}

func (s *Store) GetPlan(_ context.Context, id domain.ID) (domain.CalibrationPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	plan, ok := s.plans[id]
	if !ok {
		return domain.CalibrationPlan{}, domain.ErrNotFound
	}
	return plan, nil
}

func (s *Store) FindOpenPlan(_ context.Context, instrumentID, itemID domain.ID) (domain.CalibrationPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, plan := range s.plans {
		if plan.InstrumentID == instrumentID && plan.ItemID == itemID && plan.Status != domain.PlanCompleted && plan.Status != domain.PlanCancelled {
			return plan, nil
		}
	}
	return domain.CalibrationPlan{}, domain.ErrNotFound
}

func (s *Store) CreateExecution(_ context.Context, execution domain.CalibrationExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.executions[execution.ID]; exists {
		return domain.ErrDuplicate
	}
	s.executions[execution.ID] = execution
	root := execution.PreviousID
	if root.Empty() {
		root = execution.ID
	}
	s.executionRoots[root] = append(s.executionRoots[root], execution.ID)
	return nil
}

func (s *Store) GetExecution(_ context.Context, id domain.ID) (domain.CalibrationExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	execution, ok := s.executions[id]
	if !ok {
		return domain.CalibrationExecution{}, domain.ErrNotFound
	}
	return execution, nil
}

func (s *Store) ListExecutionVersions(_ context.Context, id domain.ID) ([]domain.CalibrationExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.executionRoots[id]
	if len(ids) == 0 {
		if execution, ok := s.executions[id]; ok && !execution.PreviousID.Empty() {
			ids = s.executionRoots[execution.PreviousID]
		}
	}
	if len(ids) == 0 {
		return nil, domain.ErrNotFound
	}
	result := make([]domain.CalibrationExecution, 0, len(ids))
	for _, executionID := range ids {
		result = append(result, s.executions[executionID])
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return result, nil
}
