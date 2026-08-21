package application

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

type CertificateInput struct {
	Number       string
	InstrumentID domain.ID
	ExecutionID  domain.ID
	SupplierID   domain.ID
	Filename     string
	ContentType  string
	Body         io.Reader
	Size         int64
	IssuedAt     time.Time
	ExpiresAt    time.Time
	CostCents    int64
	Currency     string
}

type CertificateService struct {
	repository CertificateRepository
	files      AttachmentStore
	clock      Clock
}

func NewCertificateService(repository CertificateRepository, files AttachmentStore, clock Clock) *CertificateService {
	return &CertificateService{repository: repository, files: files, clock: clock}
}

func (s *CertificateService) Create(ctx context.Context, input CertificateInput) (domain.Certificate, error) {
	if !allowedCertificateType(input.ContentType) {
		return domain.Certificate{}, domain.NewValidationError("content_type", "only PDF and common image files are accepted")
	}
	key, err := s.files.Save(ctx, input.Filename, input.ContentType, input.Body, input.Size)
	if err != nil {
		return domain.Certificate{}, err
	}
	certificate, err := domain.NewCertificate(
		input.Number, input.InstrumentID, input.ExecutionID, input.SupplierID, key,
		input.IssuedAt, input.ExpiresAt, input.CostCents, input.Currency, s.clock.Now(),
	)
	if err != nil {
		_ = s.files.Delete(ctx, key)
		return domain.Certificate{}, err
	}
	if err := s.repository.CreateCertificate(ctx, certificate); err != nil {
		_ = s.files.Delete(ctx, key)
		return domain.Certificate{}, err
	}
	return certificate, nil
}

func (s *CertificateService) Open(ctx context.Context, id domain.ID) (domain.Certificate, io.ReadCloser, error) {
	certificate, err := s.repository.GetCertificate(ctx, id)
	if err != nil {
		return domain.Certificate{}, nil, err
	}
	reader, err := s.files.Open(ctx, certificate.AttachmentKey)
	if err != nil {
		return domain.Certificate{}, nil, err
	}
	return certificate, reader, nil
}

func (s *CertificateService) ArchiveExpired(ctx context.Context, cutoff time.Time) (domain.CertificateArchive, error) {
	if cutoff.IsZero() {
		return domain.CertificateArchive{}, domain.NewValidationError("cutoff", "archive cutoff is required")
	}
	certificates, err := s.repository.ListExpiringCertificates(ctx, time.Time{}, cutoff.UTC())
	if err != nil {
		return domain.CertificateArchive{}, err
	}

	entries := make([]domain.CertificateArchiveEntry, 0, len(certificates))
	scratch := make([]byte, 64*1024)
	for _, certificate := range certificates {
		reader, err := s.files.Open(ctx, certificate.AttachmentKey)
		if err != nil {
			return domain.CertificateArchive{}, err
		}
		defer reader.Close()

		buffer := bytes.NewBuffer(scratch[:0])
		if _, err := io.CopyBuffer(buffer, reader, scratch); err != nil {
			return domain.CertificateArchive{}, err
		}
		entry, err := domain.NewCertificateArchiveEntry(certificate, archiveEntryFilename(certificate), buffer.Bytes())
		if err != nil {
			return domain.CertificateArchive{}, err
		}
		entries = append(entries, entry)
	}

	archive, err := domain.NewCertificateArchive(entries, s.clock.Now())
	if err != nil {
		return domain.CertificateArchive{}, err
	}
	payload, err := encodeCertificateArchive(archive.Entries)
	if err != nil {
		return domain.CertificateArchive{}, err
	}
	filename := "expired-certificates-" + cutoff.UTC().Format("20060102") + ".zip"
	key, err := s.files.Save(ctx, filename, "application/zip", bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return domain.CertificateArchive{}, err
	}
	if err := archive.Attach(key); err != nil {
		_ = s.files.Delete(ctx, key)
		return domain.CertificateArchive{}, err
	}
	return archive, nil
}

func archiveEntryFilename(certificate domain.Certificate) string {
	extension := strings.ToLower(filepath.Ext(certificate.AttachmentKey))
	if extension == "" {
		extension = ".bin"
	}
	name := strings.Map(func(value rune) rune {
		switch {
		case value >= 'a' && value <= 'z', value >= 'A' && value <= 'Z', value >= '0' && value <= '9', value == '-', value == '_':
			return value
		default:
			return '-'
		}
	}, certificate.Number)
	return strings.Trim(name, "-") + extension
}

func encodeCertificateArchive(entries []domain.CertificateArchiveEntry) ([]byte, error) {
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.Filename, Method: zip.Store}
		header.SetModTime(time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC))
		file, err := writer.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		if _, err := file.Write(entry.Content); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func allowedCertificateType(value string) bool {
	value = strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	switch value {
	case "application/pdf", "image/png", "image/jpeg":
		return true
	default:
		return false
	}
}
