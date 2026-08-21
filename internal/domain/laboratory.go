package domain

import (
	"strings"
	"time"
)

type Laboratory struct {
	ID        ID        `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Location  string    `json:"location"`
	ManagerID ID        `json:"manager_id"`
	Version   Version   `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewLaboratory(code, name, location string, manager ID, now time.Time) (Laboratory, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	name = strings.TrimSpace(name)
	if code == "" {
		return Laboratory{}, NewValidationError("code", "laboratory code is required")
	}
	if name == "" {
		return Laboratory{}, NewValidationError("name", "laboratory name is required")
	}
	if manager.Empty() {
		return Laboratory{}, NewValidationError("manager_id", "manager is required")
	}
	return Laboratory{
		ID: NewID("lab"), Code: code, Name: name, Location: strings.TrimSpace(location),
		ManagerID: manager, Version: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}, nil
}

func (l *Laboratory) Rename(name, location string, expected Version, now time.Time) error {
	if l.Version != expected {
		return ErrConflict
	}
	if name = strings.TrimSpace(name); name == "" {
		return NewValidationError("name", "laboratory name is required")
	}
	l.Name = name
	l.Location = strings.TrimSpace(location)
	l.Version = l.Version.Next()
	l.UpdatedAt = now.UTC()
	return nil
}
