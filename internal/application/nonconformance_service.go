package application

import (
	"context"
	"sync"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

type NonconformanceService struct {
	repository   NonconformanceRepository
	executions   CalibrationRepository
	instruments  InstrumentRepository
	transactions TransactionManager
	clock        Clock
	caseAccess   sync.RWMutex
}

type restorationAttempt struct {
	caseVersion       domain.Version
	instrumentVersion domain.Version
	authorization     domain.RestorationAuthorization
}

func prepareRestorationAttempt(nc domain.Nonconformance, instrument domain.Instrument, actor domain.Actor, comment string, expected domain.Version) (restorationAttempt, error) {
	evidence := nc.CollectRestorationEvidence()
	authorization, err := evidence.Authorize(actor, comment, expected)
	if err != nil {
		return restorationAttempt{}, err
	}
	if instrument.Status == domain.StatusDisabled {
		return restorationAttempt{}, domain.ErrInvalidTransition
	}
	return restorationAttempt{
		caseVersion:       nc.Version,
		instrumentVersion: instrument.Version,
		authorization:     authorization,
	}, nil
}

func applyRestorationAttempt(nc *domain.Nonconformance, instrument *domain.Instrument, attempt restorationAttempt, now time.Time) error {
	if nc.Version != attempt.caseVersion || instrument.Version != attempt.instrumentVersion {
		return domain.ErrConflict
	}
	if err := nc.ApplyRestoration(attempt.authorization, now); err != nil {
		return err
	}
	return instrument.ApplyStatus(domain.StatusQualified, "", attempt.instrumentVersion, now)
}

func NewNonconformanceService(repository NonconformanceRepository, executions CalibrationRepository, instruments InstrumentRepository, transactions TransactionManager, clock Clock) *NonconformanceService {
	return &NonconformanceService{repository: repository, executions: executions, instruments: instruments, transactions: transactions, clock: clock}
}

func (s *NonconformanceService) AssessImpact(ctx context.Context, id domain.ID, batches []domain.ImpactedBatch, expected domain.Version) (domain.Nonconformance, error) {
	nc, err := s.repository.GetNonconformance(ctx, id)
	if err != nil {
		return domain.Nonconformance{}, err
	}
	if err := nc.AssessImpact(batches, expected, s.clock.Now()); err != nil {
		return domain.Nonconformance{}, err
	}
	if err := s.repository.UpdateNonconformance(ctx, nc, expected); err != nil {
		return domain.Nonconformance{}, err
	}
	return nc, nil
}

func (s *NonconformanceService) RequestRetest(ctx context.Context, id domain.ID, expected domain.Version) (domain.Nonconformance, error) {
	nc, err := s.repository.GetNonconformance(ctx, id)
	if err != nil {
		return domain.Nonconformance{}, err
	}
	if err := nc.RequestRetest(expected, s.clock.Now()); err != nil {
		return domain.Nonconformance{}, err
	}
	if err := s.repository.UpdateNonconformance(ctx, nc, expected); err != nil {
		return domain.Nonconformance{}, err
	}
	return nc, nil
}

func (s *NonconformanceService) AttachRetest(ctx context.Context, id, executionID domain.ID, expected domain.Version) (domain.Nonconformance, error) {
	s.caseAccess.RLock()
	defer s.caseAccess.RUnlock()

	execution, err := s.executions.GetExecution(ctx, executionID)
	if err != nil {
		return domain.Nonconformance{}, err
	}
	nc, err := s.repository.GetNonconformance(ctx, id)
	if err != nil {
		return domain.Nonconformance{}, err
	}
	if execution.InstrumentID != nc.InstrumentID {
		return domain.Nonconformance{}, domain.NewValidationError("execution_id", "retest belongs to another instrument")
	}
	qualified := execution.Conclusion == domain.ConclusionQualified && execution.ReviewedAt != nil
	if err := nc.AttachRetest(execution.ID, qualified, expected, s.clock.Now()); err != nil {
		return domain.Nonconformance{}, err
	}
	if err := s.repository.UpdateNonconformance(ctx, nc, expected); err != nil {
		return domain.Nonconformance{}, err
	}
	return nc, nil
}

func (s *NonconformanceService) Restore(ctx context.Context, id domain.ID, actor domain.Actor, comment string, expected domain.Version) (domain.Nonconformance, error) {
	s.caseAccess.RLock()
	defer s.caseAccess.RUnlock()

	var result domain.Nonconformance
	err := s.transactions.WithinTransaction(ctx, func(tx context.Context) error {
		nc, err := s.repository.GetNonconformance(tx, id)
		if err != nil {
			return err
		}
		instrument, err := s.instruments.GetInstrument(tx, nc.InstrumentID)
		if err != nil {
			return err
		}
		attempt, err := prepareRestorationAttempt(nc, instrument, actor, comment, expected)
		if err != nil {
			return err
		}
		if err := applyRestorationAttempt(&nc, &instrument, attempt, s.clock.Now()); err != nil {
			return err
		}
		if err := s.repository.UpdateNonconformance(tx, nc, attempt.caseVersion); err != nil {
			return err
		}
		if err := s.instruments.UpdateInstrument(tx, instrument, attempt.instrumentVersion); err != nil {
			return err
		}
		result = nc
		return nil
	})
	return result, err
}
