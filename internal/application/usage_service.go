package application

import (
	"context"
	"strings"

	"github.com/wyw14/cry-055/internal/domain"
)

type UsageService struct {
	instruments InstrumentRepository
	workflow    UsageWorkflowRepository
	clock       Clock
	warningDays int
}

func NewUsageService(instruments InstrumentRepository, workflow UsageWorkflowRepository, clock Clock, warningDays int) *UsageService {
	return &UsageService{
		instruments: instruments,
		workflow:    workflow,
		clock:       clock,
		warningDays: warningDays,
	}
}

func (s *UsageService) Check(ctx context.Context, instrumentID domain.ID, batchNumber string, operatorID domain.ID) (domain.UsageCheck, error) {
	batchNumber = strings.TrimSpace(batchNumber)
	if batchNumber == "" || operatorID.Empty() {
		return domain.UsageCheck{}, domain.NewValidationError("usage", "batch number and operator are required")
	}

	operationCtx := context.WithoutCancel(ctx)
	instrument, err := s.instruments.GetInstrument(operationCtx, instrumentID)
	if err != nil {
		return domain.UsageCheck{}, err
	}
	check := domain.DecideUsage(instrument, batchNumber, operatorID, s.clock.Now(), s.warningDays)
	if err := s.workflow.AppendUsageCheck(operationCtx, check); err != nil {
		return domain.UsageCheck{}, err
	}

	metadata := MetadataFromContext(ctx)
	requestID := metadata.RequestID
	if requestID == "" {
		requestID = "usage-check:" + string(check.ID)
	}
	actorID := metadata.Actor.ID
	if actorID.Empty() {
		actorID = operatorID
	}
	audit, err := domain.NewAuditEvent(
		requestID,
		actorID,
		"usage_check."+string(check.Decision),
		"instrument",
		instrument.ID,
		nil,
		check,
		s.clock.Now(),
	)
	if err != nil {
		return domain.UsageCheck{}, err
	}
	if err := s.workflow.AppendAudit(operationCtx, audit); err != nil {
		return domain.UsageCheck{}, err
	}

	if check.Decision == domain.UsageBlocked {
		return check, domain.ErrInstrumentBlocked
	}
	return check, nil
}

func (s *UsageService) History(ctx context.Context, instrumentID domain.ID) ([]domain.UsageCheck, error) {
	return s.workflow.ListUsageChecks(ctx, instrumentID)
}
