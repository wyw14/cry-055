package domain

import (
	"strings"
	"time"
)

type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

type Alert struct {
	ID             ID            `json:"id"`
	InstrumentID   ID            `json:"instrument_id,omitempty"`
	CertificateID  ID            `json:"certificate_id,omitempty"`
	Kind           string        `json:"kind"`
	Severity       AlertSeverity `json:"severity"`
	Title          string        `json:"title"`
	Message        string        `json:"message"`
	Deduplication  string        `json:"deduplication_key"`
	AcknowledgedBy ID            `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time    `json:"acknowledged_at,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
}

func NewAlert(kind string, severity AlertSeverity, title, message, deduplication string, instrumentID, certificateID ID, now time.Time) (Alert, error) {
	kind = strings.TrimSpace(kind)
	title = strings.TrimSpace(title)
	deduplication = strings.TrimSpace(deduplication)
	if kind == "" || title == "" || deduplication == "" {
		return Alert{}, NewValidationError("alert", "kind, title and deduplication key are required")
	}
	if severity != SeverityInfo && severity != SeverityWarning && severity != SeverityCritical {
		return Alert{}, NewValidationError("severity", "unsupported severity")
	}
	return Alert{
		ID: NewID("alert"), InstrumentID: instrumentID, CertificateID: certificateID,
		Kind: kind, Severity: severity, Title: title, Message: strings.TrimSpace(message),
		Deduplication: deduplication, CreatedAt: now.UTC(),
	}, nil
}

func (a *Alert) Acknowledge(actor ID, now time.Time) error {
	if actor.Empty() {
		return NewValidationError("actor_id", "actor is required")
	}
	if a.AcknowledgedAt != nil {
		return ErrConflict
	}
	stamp := now.UTC()
	a.AcknowledgedBy = actor
	a.AcknowledgedAt = &stamp
	return nil
}
