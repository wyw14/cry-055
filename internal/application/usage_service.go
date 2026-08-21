package application

import (
	"context"
	"strings"

	"github.com/wyw14/cry-055/internal/domain"
)

type UsageService struct {
	instruments InstrumentRepository
	checks      UsageRepository
	clock       Clock
	warningDays int
}

func NewUsageService(instruments InstrumentRepository, checks UsageRepository, clock Clock, warningDays int) *UsageService {
	return &UsageService{instruments: instruments, checks: checks, clock: clock, warningDays: warningDays}
}

func (s *UsageService) Check(ctx context.Context, instrumentID domain.ID, batchNumber string, operatorID domain.ID) (domain.UsageCheck, error) {
	batchNumber = strings.TrimSpace(batchNumber)
	if batchNumber == "" || operatorID.Empty() {
		return domain.UsageCheck{}, domain.NewValidationError("usage", "batch number and operator are required")
	}
	instrument, err := s.instruments.GetInstrument(ctx, instrumentID)
	if err != nil {
		return domain.UsageCheck{}, err
	}
	check := domain.DecideUsage(instrument, batchNumber, operatorID, s.clock.Now(), s.warningDays)
	if err := s.checks.AppendUsageCheck(ctx, check); err != nil {
		return domain.UsageCheck{}, err
	}
	if check.Decision == domain.UsageBlocked {
		return check, domain.ErrInstrumentBlocked
	}
	return check, nil
}

func (s *UsageService) History(ctx context.Context, instrumentID domain.ID) ([]domain.UsageCheck, error) {
	return s.checks.ListUsageChecks(ctx, instrumentID)
}
