package localfile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	root     string
	maxBytes int64
}

func New(root string, maxBytes int64) (*Store, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if maxBytes <= 0 {
		return nil, errors.New("max attachment size must be positive")
	}
	if err := os.MkdirAll(absolute, 0o750); err != nil {
		return nil, err
	}
	return &Store{root: absolute, maxBytes: maxBytes}, nil
}

func (s *Store) Save(ctx context.Context, filename, contentType string, reader io.Reader, size int64) (string, error) {
	if size <= 0 || size > s.maxBytes {
		return "", fmt.Errorf("attachment size outside allowed range")
	}
	safeName := filepath.Base(strings.TrimSpace(filename))
	if safeName == "." || safeName == "" {
		return "", errors.New("attachment filename is required")
	}
	hash := sha256.New()
	limited := io.LimitReader(reader, s.maxBytes+1)
	temporary, err := os.CreateTemp(s.root, "upload-*.tmp")
	if err != nil {
		return "", err
	}
	temporaryName := temporary.Name()
	defer func() { temporary.Close(); os.Remove(temporaryName) }()
	written, err := io.Copy(io.MultiWriter(temporary, hash), limited)
	if err != nil {
		return "", err
	}
	if written != size || written > s.maxBytes {
		return "", errors.New("attachment size mismatch")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := temporary.Sync(); err != nil {
		return "", err
	}
	if err := temporary.Close(); err != nil {
		return "", err
	}
	key := hex.EncodeToString(hash.Sum(nil)) + strings.ToLower(filepath.Ext(safeName))
	target := filepath.Join(s.root, key)
	if err := os.Rename(temporaryName, target); err != nil {
		if errors.Is(err, os.ErrExist) {
			return key, nil
		}
		return "", err
	}
	return key, nil
}

func (s *Store) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, err := s.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}
func (s *Store) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.path(key)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
func (s *Store) path(key string) (string, error) {
	key = filepath.Base(strings.TrimSpace(key))
	if key == "" || key == "." {
		return "", errors.New("invalid attachment key")
	}
	target := filepath.Join(s.root, key)
	relative, err := filepath.Rel(s.root, target)
	if err != nil || strings.HasPrefix(relative, "..") {
		return "", errors.New("attachment key escapes storage root")
	}
	return target, nil
}
