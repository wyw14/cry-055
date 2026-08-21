package application

import (
	"context"
	"encoding/csv"
	"io"
	"strconv"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

type InstrumentTrace struct {
	Instrument domain.Instrument             `json:"instrument"`
	Audits     []domain.AuditEvent           `json:"audits"`
	Usage      []domain.UsageCheck           `json:"usage_checks"`
	Executions []domain.CalibrationExecution `json:"executions"`
}

type ReportService struct {
	instruments InstrumentRepository
	audits      AuditRepository
	usage       UsageRepository
	executions  CalibrationRepository
}

func NewReportService(instruments InstrumentRepository, audits AuditRepository, usage UsageRepository, executions CalibrationRepository) *ReportService {
	return &ReportService{instruments: instruments, audits: audits, usage: usage, executions: executions}
}

func (s *ReportService) InstrumentTrace(ctx context.Context, instrumentID domain.ID) (InstrumentTrace, error) {
	instrument, err := s.instruments.GetInstrument(ctx, instrumentID)
	if err != nil {
		return InstrumentTrace{}, err
	}
	audits, err := s.audits.ListAudit(ctx, "instrument", instrumentID)
	if err != nil {
		return InstrumentTrace{}, err
	}
	checks, err := s.usage.ListUsageChecks(ctx, instrumentID)
	if err != nil {
		return InstrumentTrace{}, err
	}
	return InstrumentTrace{Instrument: instrument, Audits: audits, Usage: checks}, nil
}

func (s *ReportService) ExportInstrumentCSV(ctx context.Context, page domain.PageRequest, writer io.Writer) error {
	instruments, err := s.instruments.ListInstruments(ctx, page)
	if err != nil {
		return err
	}
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write([]string{"asset_number", "name", "model", "criticality", "status", "next_due_at", "version"}); err != nil {
		return err
	}
	for _, instrument := range instruments.Items {
		row := []string{
			instrument.AssetNumber, instrument.Name, instrument.Model, string(instrument.Criticality),
			string(instrument.Status), instrument.NextDueAt.Format(time.RFC3339), strconv.FormatInt(int64(instrument.Version), 10),
		}
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}
