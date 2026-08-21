package application

import (
	"context"
	"io"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

type Clock interface {
	Now() time.Time
}

type InstrumentRepository interface {
	CreateInstrument(context.Context, domain.Instrument) error
	UpdateInstrument(context.Context, domain.Instrument, domain.Version) error
	GetInstrument(context.Context, domain.ID) (domain.Instrument, error)
	GetInstrumentByAssetNumber(context.Context, string) (domain.Instrument, error)
	ListInstruments(context.Context, domain.PageRequest) (domain.Page[domain.Instrument], error)
}

type LaboratoryRepository interface {
	CreateLaboratory(context.Context, domain.Laboratory) error
	GetLaboratory(context.Context, domain.ID) (domain.Laboratory, error)
}

type CalibrationRepository interface {
	CreateItem(context.Context, domain.CalibrationItem) error
	GetItem(context.Context, domain.ID) (domain.CalibrationItem, error)
	CreatePlan(context.Context, domain.CalibrationPlan) error
	UpdatePlan(context.Context, domain.CalibrationPlan, domain.Version) error
	GetPlan(context.Context, domain.ID) (domain.CalibrationPlan, error)
	FindOpenPlan(context.Context, domain.ID, domain.ID) (domain.CalibrationPlan, error)
	CreateExecution(context.Context, domain.CalibrationExecution) error
	UpdateExecution(context.Context, domain.CalibrationExecution, domain.Version) error
	GetExecution(context.Context, domain.ID) (domain.CalibrationExecution, error)
	ListExecutionVersions(context.Context, domain.ID) ([]domain.CalibrationExecution, error)
}

type NonconformanceRepository interface {
	CreateNonconformance(context.Context, domain.Nonconformance) error
	UpdateNonconformance(context.Context, domain.Nonconformance, domain.Version) error
	GetNonconformance(context.Context, domain.ID) (domain.Nonconformance, error)
	FindOpenByInstrument(context.Context, domain.ID) (domain.Nonconformance, error)
}

type CertificateRepository interface {
	CreateCertificate(context.Context, domain.Certificate) error
	GetCertificate(context.Context, domain.ID) (domain.Certificate, error)
	ListExpiringCertificates(context.Context, time.Time, time.Time) ([]domain.Certificate, error)
}

type AlertRepository interface {
	UpsertAlert(context.Context, domain.Alert) (domain.Alert, bool, error)
	AcknowledgeAlert(context.Context, domain.Alert) error
	ListOpenAlerts(context.Context, domain.PageRequest) (domain.Page[domain.Alert], error)
}

type AuditRepository interface {
	AppendAudit(context.Context, domain.AuditEvent) error
	ListAudit(context.Context, string, domain.ID) ([]domain.AuditEvent, error)
}

type UsageRepository interface {
	AppendUsageCheck(context.Context, domain.UsageCheck) error
	ListUsageChecks(context.Context, domain.ID) ([]domain.UsageCheck, error)
}

type TransactionManager interface {
	WithinTransaction(context.Context, func(context.Context) error) error
}

type IdempotencyRepository interface {
	Reserve(context.Context, string, string, time.Duration) (bool, error)
	Complete(context.Context, string, string) error
	Release(context.Context, string, string) error
}

type AttachmentStore interface {
	Save(context.Context, string, string, io.Reader, int64) (string, error)
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}

type Notifier interface {
	Notify(context.Context, domain.Alert) error
}

type RequestMetadata struct {
	RequestID string
	Actor     domain.Actor
}

type metadataKey struct{}

func WithRequestMetadata(ctx context.Context, value RequestMetadata) context.Context {
	return context.WithValue(ctx, metadataKey{}, value)
}

func MetadataFromContext(ctx context.Context) RequestMetadata {
	value, _ := ctx.Value(metadataKey{}).(RequestMetadata)
	return value
}
