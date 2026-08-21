package localfile

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestStoreRoundTripAndContentAddressing(t *testing.T) {
	store, err := New(t.TempDir(), 1024)
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("certificate")
	key, err := store.Save(context.Background(), "certificate.pdf", "application/pdf", bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatal(err)
	}
	reader, err := store.Open(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	actual, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, content) {
		t.Fatalf("want %q got %q", content, actual)
	}
}
func TestStoreRejectsSizeMismatch(t *testing.T) {
	store, _ := New(t.TempDir(), 1024)
	if _, err := store.Save(context.Background(), "certificate.pdf", "application/pdf", bytes.NewBufferString("small"), 999); err == nil {
		t.Fatal("expected size mismatch")
	}
}
