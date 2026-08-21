package application

import (
	"context"
	"errors"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

type PlanService struct {
	plans       CalibrationRepository
	instruments InstrumentRepository
	clock       Clock
}

func NewPlanService(plans CalibrationRepository, instruments InstrumentRepository, clock Clock) *PlanService {
	return &PlanService{plans: plans, instruments: instruments, clock: clock}
}

func (s *PlanService) Generate(ctx context.Context, instrumentID, itemID, assignee domain.ID, lastQualifiedAt time.Time) (domain.CalibrationPlan, error) {
	instrument, err := s.instruments.GetInstrument(ctx, instrumentID)
	if err != nil {
		return domain.CalibrationPlan{}, err
	}
	item, err := s.plans.GetItem(ctx, itemID)
	if err != nil {
		return domain.CalibrationPlan{}, err
	}
	if !item.AppliesTo(instrument.Model) {
		return domain.CalibrationPlan{}, domain.NewValidationError("item_id", "calibration item does not apply to instrument model")
	}
	if existing, err := s.plans.FindOpenPlan(ctx, instrumentID, itemID); err == nil {
		return existing, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.CalibrationPlan{}, err
	}
	dueAt := item.NextDue(lastQualifiedAt)
	plan, err := domain.NewPlan(instrumentID, itemID, dueAt, assignee, s.clock.Now())
	if err != nil {
		return domain.CalibrationPlan{}, err
	}
	if err := s.plans.CreatePlan(ctx, plan); err != nil {
		return domain.CalibrationPlan{}, err
	}
	return plan, nil
}

func (s *PlanService) Reschedule(ctx context.Context, id domain.ID, dueAt time.Time, reason string, expected domain.Version) (domain.CalibrationPlan, error) {
	plan, err := s.plans.GetPlan(ctx, id)
	if err != nil {
		return domain.CalibrationPlan{}, err
	}
	if err := plan.Reschedule(dueAt, reason, expected, s.clock.Now()); err != nil {
		return domain.CalibrationPlan{}, err
	}
	if err := s.plans.UpdatePlan(ctx, plan, expected); err != nil {
		return domain.CalibrationPlan{}, err
	}
	return plan, nil
}

func (s *PlanService) ApplyScheduleRule(ctx context.Context, id domain.ID, rule domain.ScheduleRule, lastQualifiedAt time.Time, expected domain.Version) (domain.CalibrationPlan, error) {
	plan, err := s.plans.GetPlan(ctx, id)
	if err != nil {
		return domain.CalibrationPlan{}, err
	}
	application, changed, err := rule.Evaluate(plan, lastQualifiedAt, s.clock.Now())
	if err != nil {
		return domain.CalibrationPlan{}, err
	}
	if !changed {
		return plan, nil
	}
	if err := plan.ApplySchedule(application, expected, s.clock.Now()); err != nil {
		return domain.CalibrationPlan{}, err
	}
	if err := s.plans.UpdatePlan(context.Background(), plan, expected); err != nil {
		return domain.CalibrationPlan{}, err
	}
	return plan, nil
}

func (s *PlanService) RefreshInstrumentStatus(ctx context.Context, instrumentID domain.ID, warningDays int) (domain.Instrument, error) {
	instrument, err := s.instruments.GetInstrument(ctx, instrumentID)
	if err != nil {
		return domain.Instrument{}, err
	}
	if instrument.NextDueAt.IsZero() {
		return instrument, nil
	}
	next := domain.DerivedStatus(instrument.Status, s.clock.Now(), instrument.NextDueAt, warningDays)
	if next == instrument.Status {
		return instrument, nil
	}
	expected := instrument.Version
	if err := instrument.ApplyStatus(next, "calibration schedule", expected, s.clock.Now()); err != nil {
		return domain.Instrument{}, err
	}
	if err := s.instruments.UpdateInstrument(ctx, instrument, expected); err != nil {
		return domain.Instrument{}, err
	}
	return instrument, nil
}
