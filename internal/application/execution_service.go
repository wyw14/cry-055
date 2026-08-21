package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

type ExecutionInput struct {
	InstrumentID  domain.ID
	ItemID        domain.ID
	PlanID        domain.ID
	ExecutorID    domain.ID
	Measurements  []domain.Measurement
	CertificateID domain.ID
	CompletedAt   time.Time
}

type ExecutionService struct {
	repository   CalibrationRepository
	instruments  InstrumentRepository
	nonconform   NonconformanceRepository
	transactions TransactionManager
	audits       AuditRepository
	clock        Clock
}

func NewExecutionService(repository CalibrationRepository, instruments InstrumentRepository, nonconform NonconformanceRepository, transactions TransactionManager, audits AuditRepository, clock Clock) *ExecutionService {
	return &ExecutionService{repository: repository, instruments: instruments, nonconform: nonconform, transactions: transactions, audits: audits, clock: clock}
}

func (s *ExecutionService) Record(ctx context.Context, input ExecutionInput) (domain.CalibrationExecution, error) {
	var result domain.CalibrationExecution
	err := s.transactions.WithinTransaction(ctx, func(tx context.Context) error {
		instrument, err := s.instruments.GetInstrument(tx, input.InstrumentID)
		if err != nil {
			return err
		}
		storedInstrumentVersion := instrument.Version
		item, err := s.repository.GetItem(tx, input.ItemID)
		if err != nil {
			return err
		}
		plan, err := s.repository.GetPlan(tx, input.PlanID)
		if err != nil {
			return err
		}
		if plan.InstrumentID != input.InstrumentID || plan.ItemID != input.ItemID {
			return domain.NewValidationError("plan_id", "plan does not match instrument and item")
		}
		for index, measurement := range input.Measurements {
			if measurement.Error > item.Tolerance && measurement.Passed {
				return domain.NewValidationError("measurements", fmt.Sprintf("measurement %d has inconsistent result", index))
			}
		}
		execution, err := domain.NewExecution(input.InstrumentID, input.ItemID, input.PlanID, input.ExecutorID, input.Measurements, input.CertificateID, input.CompletedAt, s.clock.Now())
		if err != nil {
			return err
		}
		if err := s.repository.CreateExecution(tx, execution); err != nil {
			return err
		}
		planVersion := plan.Version
		if err := plan.Complete(planVersion, s.clock.Now()); err != nil {
			return err
		}
		if err := s.repository.UpdatePlan(tx, plan, planVersion); err != nil {
			return err
		}
		instrumentVersion := instrument.Version
		if execution.Conclusion == domain.ConclusionQualified {
			nextDue := item.NextDue(execution.CompletedAt)
			if err := instrument.Schedule(nextDue, instrumentVersion, s.clock.Now()); err != nil {
				return err
			}
			instrumentVersion = instrument.Version
			if instrument.Status != domain.StatusQualified {
				if err := instrument.ApplyStatus(domain.StatusQualified, "", instrumentVersion, s.clock.Now()); err != nil {
					return err
				}
			}
		} else {
			if err := instrument.ApplyStatus(domain.StatusUnqualified, "calibration result outside tolerance", instrumentVersion, s.clock.Now()); err != nil {
				return err
			}
			nc, err := domain.NewNonconformance(instrument.ID, execution.ID, "calibration result outside tolerance", s.clock.Now())
			if err != nil {
				return err
			}
			if err := s.nonconform.CreateNonconformance(tx, nc); err != nil {
				return err
			}
		}
		if err := s.instruments.UpdateInstrument(tx, instrument, storedInstrumentVersion); err != nil {
			return err
		}
		result = execution
		return nil
	})
	return result, err
}

func (s *ExecutionService) Revise(ctx context.Context, executionID, actorID domain.ID, measurements []domain.Measurement) (domain.CalibrationExecution, error) {
	current, err := s.repository.GetExecution(ctx, executionID)
	if err != nil {
		return domain.CalibrationExecution{}, err
	}
	revised, err := current.Revise(measurements, actorID, s.clock.Now())
	if err != nil {
		return domain.CalibrationExecution{}, err
	}
	if err := s.repository.CreateExecution(ctx, revised); err != nil {
		return domain.CalibrationExecution{}, err
	}
	return revised, nil
}

func (s *ExecutionService) Review(ctx context.Context, executionID, reviewerID domain.ID, comment string) (domain.CalibrationExecution, error) {
	execution, err := s.repository.GetExecution(ctx, executionID)
	if err != nil {
		return domain.CalibrationExecution{}, domain.NewReviewFailure(executionID, "load", err)
	}
	expected := execution.Version
	if err := execution.Review(reviewerID, comment, s.clock.Now()); err != nil {
		return domain.CalibrationExecution{}, domain.NewReviewFailure(executionID, "decision", err)
	}
	if err := s.repository.UpdateExecution(context.Background(), execution, expected); err != nil {
		return domain.CalibrationExecution{}, domain.NewReviewFailure(executionID, "execution update", err)
	}
	metadata := MetadataFromContext(ctx)
	event, err := domain.NewAuditEvent(metadata.RequestID, reviewerID, "calibration_execution.review", "calibration_execution", execution.ID, nil, execution, s.clock.Now())
	if err != nil {
		return domain.CalibrationExecution{}, domain.NewReviewFailure(executionID, "audit creation", err)
	}
	err = s.transactions.WithinTransaction(ctx, func(tx context.Context) error {
		return s.audits.AppendAudit(tx, event)
	})
	if err != nil {
		return domain.CalibrationExecution{}, domain.NewReviewFailure(executionID, "audit append", err)
	}
	return execution, nil
}
