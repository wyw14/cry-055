package domain

import (
	"crypto/sha256"
	"encoding/hex"
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

type CertificateArchiveEntry struct {
	CertificateID ID     `json:"certificate_id"`
	Number        string `json:"number"`
	Filename      string `json:"filename"`
	Size          int64  `json:"size"`
	SHA256        string `json:"sha256"`
	Content       []byte `json:"-"`
}

func NewCertificateArchiveEntry(certificate Certificate, filename string, content []byte) (CertificateArchiveEntry, error) {
	filename = strings.TrimSpace(filename)
	if certificate.ID.Empty() || filename == "" {
		return CertificateArchiveEntry{}, NewValidationError("archive_entry", "certificate and filename are required")
	}
	if len(content) == 0 {
		return CertificateArchiveEntry{}, NewValidationError("archive_entry", "certificate content is empty")
	}
	digest := sha256.Sum256(content)
	return CertificateArchiveEntry{
		CertificateID: certificate.ID,
		Number:        certificate.Number,
		Filename:      filename,
		Size:          int64(len(content)),
		SHA256:        hex.EncodeToString(digest[:]),
		Content:       content,
	}, nil
}

type CertificateArchive struct {
	ID            ID                        `json:"id"`
	AttachmentKey string                    `json:"attachment_key"`
	Entries       []CertificateArchiveEntry `json:"entries"`
	CreatedAt     time.Time                 `json:"created_at"`
}

func NewCertificateArchive(entries []CertificateArchiveEntry, now time.Time) (CertificateArchive, error) {
	if len(entries) == 0 {
		return CertificateArchive{}, NewValidationError("archive", "at least one expired certificate is required")
	}
	return CertificateArchive{ID: NewID("certarc"), Entries: entries, CreatedAt: now.UTC()}, nil
}

func (a *CertificateArchive) Attach(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return NewValidationError("attachment_key", "archive attachment is required")
	}
	a.AttachmentKey = key
	return nil
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
