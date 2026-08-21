package application

import (
	"context"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

type CalibrationItemInput struct {
	Code                string
	Name                string
	PeriodDays          int
	WarningDays         int
	Tolerance           float64
	Unit                string
	ReferenceStandardID domain.ID
	ApplicableModels    []string
}

type CatalogService struct {
	labs         LaboratoryRepository
	calibrations CalibrationRepository
	clock        Clock
}

func NewCatalogService(labs LaboratoryRepository, calibrations CalibrationRepository, clock Clock) *CatalogService {
	return &CatalogService{labs: labs, calibrations: calibrations, clock: clock}
}
func (s *CatalogService) CreateLaboratory(ctx context.Context, code, name, location string, managerID domain.ID) (domain.Laboratory, error) {
	value, err := domain.NewLaboratory(code, name, location, managerID, s.clock.Now())
	if err != nil {
		return domain.Laboratory{}, err
	}
	if err := s.labs.CreateLaboratory(ctx, value); err != nil {
		return domain.Laboratory{}, err
	}
	return value, nil
}
func (s *CatalogService) CreateCalibrationItem(ctx context.Context, input CalibrationItemInput) (domain.CalibrationItem, error) {
	value, err := domain.NewCalibrationItem(input.Code, input.Name, input.PeriodDays, input.WarningDays, input.Tolerance, input.Unit, input.ReferenceStandardID, input.ApplicableModels, s.clock.Now())
	if err != nil {
		return domain.CalibrationItem{}, err
	}
	if err := s.calibrations.CreateItem(ctx, value); err != nil {
		return domain.CalibrationItem{}, err
	}
	return value, nil
}
func (s *CatalogService) DueWindow(ctx context.Context, itemID, instrumentID domain.ID, lastQualifiedAt, timeNow time.Time) (domain.DueWindow, error) {
	item, err := s.calibrations.GetItem(ctx, itemID)
	if err != nil {
		return domain.DueWindow{}, err
	}
	rule := domain.ScheduleRule{InstrumentID: instrumentID, ItemID: itemID, PeriodDays: item.PeriodDays, WarningDays: item.WarningDays, EffectiveAt: lastQualifiedAt}
	return domain.BuildDueWindow(rule, lastQualifiedAt, timeNow)
}
