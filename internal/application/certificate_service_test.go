package application

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
	"github.com/wyw14/cry-055/internal/repository/memory"
)

type archiveAttachmentStore struct {
	files     map[string][]byte
	active    int
	maxActive int
}

func (s *archiveAttachmentStore) Save(_ context.Context, filename, _ string, reader io.Reader, _ int64) (string, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	key := filename
	s.files[key] = append([]byte(nil), content...)
	return key, nil
}

func (s *archiveAttachmentStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	content, ok := s.files[key]
	if !ok {
		return nil, errors.New("attachment not found")
	}
	s.active++
	if s.active > s.maxActive {
		s.maxActive = s.active
	}
	return &trackedArchiveReader{Reader: bytes.NewReader(content), close: func() { s.active-- }}, nil
}

func (s *archiveAttachmentStore) Delete(_ context.Context, key string) error {
	delete(s.files, key)
	return nil
}

type trackedArchiveReader struct {
	*bytes.Reader
	close func()
}

func (r *trackedArchiveReader) Close() error {
	r.close()
	return nil
}

func TestArchiveExpiredCertificatesPreservesAttachmentsAndClosesReaders(t *testing.T) {
	now := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	store := memory.New()
	files := &archiveAttachmentStore{files: map[string][]byte{
		"first.pdf":  []byte("first-certificate-payload"),
		"second.pdf": []byte("second-certificate-payload-is-longer"),
	}}
	first, err := domain.NewCertificate("CERT-A", "instrument-a", "execution-a", "supplier-a", "first.pdf", now.Add(-365*24*time.Hour), now.Add(-48*time.Hour), 1200, "CNY", now.Add(-365*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	second, err := domain.NewCertificate("CERT-B", "instrument-b", "execution-b", "supplier-b", "second.pdf", now.Add(-300*24*time.Hour), now.Add(-24*time.Hour), 1800, "CNY", now.Add(-300*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateCertificate(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateCertificate(context.Background(), second); err != nil {
		t.Fatal(err)
	}

	service := NewCertificateService(store, files, fixedClock{value: now})
	archive, err := service.ArchiveExpired(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if files.maxActive != 1 || files.active != 0 {
		t.Errorf("attachment readers were not released one at a time: peak=%d active=%d", files.maxActive, files.active)
	}
	payload := files.files[archive.AttachmentKey]
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatal(err)
	}
	contents := make(map[string]string, len(reader.File))
	for _, file := range reader.File {
		entry, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, readErr := io.ReadAll(entry)
		closeErr := entry.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("read archive entry: read=%v close=%v", readErr, closeErr)
		}
		contents[file.Name] = string(content)
	}
	if contents["CERT-A.pdf"] != "first-certificate-payload" || contents["CERT-B.pdf"] != "second-certificate-payload-is-longer" {
		t.Fatalf("certificate archive mixed attachment contents: %s", strings.Join([]string{contents["CERT-A.pdf"], contents["CERT-B.pdf"]}, " | "))
	}
}
