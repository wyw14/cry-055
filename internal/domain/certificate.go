package domain

import (
	"strings"
	"time"
)

type Certificate struct {
	ID            ID        `json:"id"`
	Number        string    `json:"number"`
	InstrumentID  ID        `json:"instrument_id"`
	ExecutionID   ID        `json:"execution_id"`
	SupplierID    ID        `json:"supplier_id"`
	AttachmentKey string    `json:"attachment_key"`
	IssuedAt      time.Time `json:"issued_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	CostCents     int64     `json:"cost_cents"`
	Currency      string    `json:"currency"`
	Version       Version   `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
}

func NewCertificate(number string, instrumentID, executionID, supplierID ID, attachmentKey string, issuedAt, expiresAt time.Time, costCents int64, currency string, now time.Time) (Certificate, error) {
	number = strings.TrimSpace(number)
	attachmentKey = strings.TrimSpace(attachmentKey)
	if number == "" || instrumentID.Empty() || executionID.Empty() || supplierID.Empty() {
		return Certificate{}, NewValidationError("certificate", "number and references are required")
	}
	if attachmentKey == "" {
		return Certificate{}, NewValidationError("attachment_key", "certificate attachment is required")
	}
	if issuedAt.IsZero() || !expiresAt.After(issuedAt) {
		return Certificate{}, NewValidationError("expires_at", "expiry must be after issue time")
	}
	if costCents < 0 {
		return Certificate{}, NewValidationError("cost_cents", "cost cannot be negative")
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if len(currency) != 3 {
		return Certificate{}, NewValidationError("currency", "currency must be an ISO code")
	}
	return Certificate{
		ID: NewID("cert"), Number: number, InstrumentID: instrumentID, ExecutionID: executionID,
		SupplierID: supplierID, AttachmentKey: attachmentKey, IssuedAt: issuedAt.UTC(),
		ExpiresAt: expiresAt.UTC(), CostCents: costCents, Currency: currency,
		Version: 1, CreatedAt: now.UTC(),
	}, nil
}

func (c Certificate) ExpiringWithin(now time.Time, days int) bool {
	if days < 0 || !c.ExpiresAt.After(now) {
		return false
	}
	return c.ExpiresAt.Sub(now) <= time.Duration(days)*24*time.Hour
}

type Supplier struct {
	ID        ID        `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Contact   string    `json:"contact"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Qualified bool      `json:"qualified"`
	ValidTo   time.Time `json:"valid_to"`
	Version   Version   `json:"version"`
}

func (s Supplier) CanCalibrate(at time.Time) bool {
	return s.Qualified && s.ValidTo.After(at)
}
