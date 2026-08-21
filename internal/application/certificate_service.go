package application

import (
	"context"
	"io"
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

func allowedCertificateType(value string) bool {
	value = strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	switch value {
	case "application/pdf", "image/png", "image/jpeg":
		return true
	default:
		return false
	}
}
