package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

type AlertService struct {
	alerts       AlertRepository
	certificates CertificateRepository
	instruments  InstrumentRepository
	notifier     Notifier
	clock        Clock
}

func NewAlertService(alerts AlertRepository, certificates CertificateRepository, instruments InstrumentRepository, notifier Notifier, clock Clock) *AlertService {
	return &AlertService{alerts: alerts, certificates: certificates, instruments: instruments, notifier: notifier, clock: clock}
}

func (s *AlertService) ScanInstrument(ctx context.Context, id domain.ID, warningDays int) (domain.Alert, bool, error) {
	instrument, err := s.instruments.GetInstrument(ctx, id)
	if err != nil {
		return domain.Alert{}, false, err
	}
	status := domain.DerivedStatus(instrument.Status, s.clock.Now(), instrument.NextDueAt, warningDays)
	if status != domain.StatusDueSoon && status != domain.StatusOverdue && status != domain.StatusUnqualified && status != domain.StatusDisabled {
		return domain.Alert{}, false, nil
	}
	severity := domain.SeverityWarning
	if status == domain.StatusOverdue || status == domain.StatusUnqualified || instrument.Criticality == domain.CriticalityCritical {
		severity = domain.SeverityCritical
	}
	alert, err := domain.NewAlert(
		"instrument_status", severity, "Instrument requires attention",
		fmt.Sprintf("asset %s is %s", instrument.AssetNumber, status),
		fmt.Sprintf("instrument:%s:%s:%s", id, status, instrument.NextDueAt.Format("2006-01-02")),
		id, "", s.clock.Now(),
	)
	if err != nil {
		return domain.Alert{}, false, err
	}
	stored, err := s.alerts.FindAlertByDeduplication(ctx, alert.Deduplication)
	if err == nil {
		return stored, false, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.Alert{}, false, err
	}
	audit, err := alertScanAudit(alert, s.clock.Now())
	if err != nil {
		return domain.Alert{}, false, err
	}
	if err := s.alerts.RecordAlertScan(ctx, alert, audit); err != nil {
		return domain.Alert{}, false, err
	}
	_ = s.notifier.Notify(ctx, alert)
	return alert, true, nil
}

func (s *AlertService) ScanCertificates(ctx context.Context, horizonDays int) ([]domain.Alert, error) {
	now := s.clock.Now()
	certificates, err := s.certificates.ListExpiringCertificates(ctx, now, now.Add(time.Duration(horizonDays)*24*time.Hour))
	if err != nil {
		return nil, err
	}
	result := make([]domain.Alert, 0, len(certificates))
	for _, certificate := range certificates {
		alert, err := domain.NewAlert(
			"certificate_expiry", domain.SeverityWarning, "Calibration certificate is expiring",
			fmt.Sprintf("certificate %s expires on %s", certificate.Number, certificate.ExpiresAt.Format("2006-01-02")),
			fmt.Sprintf("certificate:%s:%s", certificate.ID, certificate.ExpiresAt.Format("2006-01-02")),
			certificate.InstrumentID, certificate.ID, now,
		)
		if err != nil {
			return nil, err
		}
		stored, err := s.alerts.FindAlertByDeduplication(ctx, alert.Deduplication)
		if err == nil {
			result = append(result, stored)
			continue
		}
		if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		audit, err := alertScanAudit(alert, now)
		if err != nil {
			return nil, err
		}
		if err := s.alerts.RecordAlertScan(ctx, alert, audit); err != nil {
			return nil, err
		}
		_ = s.notifier.Notify(ctx, alert)
		result = append(result, alert)
	}
	return result, nil
}

func alertScanAudit(alert domain.Alert, now time.Time) (domain.AuditEvent, error) {
	return domain.NewAuditEvent(
		"alert-scan:"+alert.Deduplication,
		"system-alert-scanner",
		"alert.created",
		"instrument",
		alert.InstrumentID,
		nil,
		alert,
		now,
	)
}

func (s *AlertService) List(ctx context.Context, page domain.PageRequest) (domain.Page[domain.Alert], error) {
	normalized, err := page.Normalize(
		map[string]struct{}{"created_at": {}, "severity": {}, "kind": {}},
		map[string]struct{}{"severity": {}, "kind": {}, "instrument_id": {}},
	)
	if err != nil {
		return domain.Page[domain.Alert]{}, err
	}
	return s.alerts.ListOpenAlerts(ctx, normalized)
}
