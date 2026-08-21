package application

import (
	"context"
	"errors"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

type InstrumentService struct {
	instruments InstrumentRepository
	labs        LaboratoryRepository
	audits      AuditRepository
	clock       Clock
}

func NewInstrumentService(instruments InstrumentRepository, labs LaboratoryRepository, audits AuditRepository, clock Clock) *InstrumentService {
	return &InstrumentService{instruments: instruments, labs: labs, audits: audits, clock: clock}
}

func (s *InstrumentService) Register(ctx context.Context, input domain.InstrumentInput) (domain.Instrument, error) {
	if _, err := s.labs.GetLaboratory(ctx, input.LaboratoryID); err != nil {
		return domain.Instrument{}, err
	}
	if existing, err := s.instruments.GetInstrumentByAssetNumber(ctx, input.AssetNumber); err == nil && !existing.ID.Empty() {
		return domain.Instrument{}, domain.ErrDuplicate
	} else if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.Instrument{}, err
	}
	instrument, err := domain.NewInstrument(input, s.clock.Now())
	if err != nil {
		return domain.Instrument{}, err
	}
	if err := s.instruments.CreateInstrument(ctx, instrument); err != nil {
		return domain.Instrument{}, err
	}
	s.audit(ctx, "instrument.register", instrument.ID, nil, instrument)
	return instrument, nil
}

func (s *InstrumentService) Schedule(ctx context.Context, id domain.ID, dueAt time.Time, expected domain.Version) (domain.Instrument, error) {
	instrument, err := s.instruments.GetInstrument(ctx, id)
	if err != nil {
		return domain.Instrument{}, err
	}
	before := instrument
	if err := instrument.Schedule(dueAt, expected, s.clock.Now()); err != nil {
		return domain.Instrument{}, err
	}
	if err := s.instruments.UpdateInstrument(ctx, instrument, expected); err != nil {
		return domain.Instrument{}, err
	}
	s.audit(ctx, "instrument.schedule", id, before, instrument)
	return instrument, nil
}

func (s *InstrumentService) ChangeStatus(ctx context.Context, id domain.ID, status domain.InstrumentStatus, reason string, expected domain.Version) (domain.Instrument, error) {
	instrument, err := s.instruments.GetInstrument(ctx, id)
	if err != nil {
		return domain.Instrument{}, err
	}
	before := instrument
	if err := instrument.ApplyStatus(status, reason, expected, s.clock.Now()); err != nil {
		return domain.Instrument{}, err
	}
	if err := s.instruments.UpdateInstrument(ctx, instrument, expected); err != nil {
		return domain.Instrument{}, err
	}
	s.audit(ctx, "instrument.status_changed", id, before, instrument)
	return instrument, nil
}

func (s *InstrumentService) Get(ctx context.Context, id domain.ID) (domain.Instrument, error) {
	instrument, err := s.instruments.GetInstrument(ctx, id)
	if err != nil {
		return domain.Instrument{}, err
	}
	if !instrument.NextDueAt.IsZero() {
		instrument.Status = domain.DerivedStatus(instrument.Status, s.clock.Now(), instrument.NextDueAt, 30)
	}
	return instrument, nil
}

func (s *InstrumentService) List(ctx context.Context, page domain.PageRequest) (domain.Page[domain.Instrument], error) {
	normalized, err := page.Normalize(
		map[string]struct{}{"created_at": {}, "asset_number": {}, "next_due_at": {}, "status": {}},
		map[string]struct{}{"laboratory_id": {}, "status": {}, "criticality": {}, "owner_id": {}},
	)
	if err != nil {
		return domain.Page[domain.Instrument]{}, err
	}
	if normalized.Filters["status"] == "" {
		normalized.AsOf = s.clock.Now().UTC()
	}
	return s.instruments.ListInstruments(ctx, normalized)
}

func (s *InstrumentService) audit(ctx context.Context, action string, id domain.ID, before, after any) {
	metadata := MetadataFromContext(ctx)
	if metadata.Actor.ID.Empty() || metadata.RequestID == "" {
		return
	}
	event, err := domain.NewAuditEvent(metadata.RequestID, metadata.Actor.ID, action, "instrument", id, before, after, s.clock.Now())
	if err == nil {
		_ = s.audits.AppendAudit(ctx, event)
	}
}
