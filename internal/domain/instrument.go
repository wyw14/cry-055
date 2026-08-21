package domain

import (
	"strings"
	"time"
)

type Criticality string

const (
	CriticalityNormal   Criticality = "normal"
	CriticalityMajor    Criticality = "major"
	CriticalityCritical Criticality = "critical"
)

func (c Criticality) Valid() bool {
	return c == CriticalityNormal || c == CriticalityMajor || c == CriticalityCritical
}

type Instrument struct {
	ID             ID               `json:"id"`
	LaboratoryID   ID               `json:"laboratory_id"`
	AssetNumber    string           `json:"asset_number"`
	Name           string           `json:"name"`
	Model          string           `json:"model"`
	SerialNumber   string           `json:"serial_number"`
	Location       string           `json:"location"`
	OwnerID        ID               `json:"owner_id"`
	Criticality    Criticality      `json:"criticality"`
	Status         InstrumentStatus `json:"status"`
	NextDueAt      time.Time        `json:"next_due_at"`
	DisabledReason string           `json:"disabled_reason,omitempty"`
	Version        Version          `json:"version"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type InstrumentInput struct {
	LaboratoryID ID
	AssetNumber  string
	Name         string
	Model        string
	SerialNumber string
	Location     string
	OwnerID      ID
	Criticality  Criticality
}

func NewInstrument(input InstrumentInput, now time.Time) (Instrument, error) {
	input.AssetNumber = strings.ToUpper(strings.TrimSpace(input.AssetNumber))
	input.Name = strings.TrimSpace(input.Name)
	if input.LaboratoryID.Empty() {
		return Instrument{}, NewValidationError("laboratory_id", "laboratory is required")
	}
	if input.AssetNumber == "" || input.Name == "" {
		return Instrument{}, NewValidationError("asset_number", "asset number and name are required")
	}
	if input.OwnerID.Empty() {
		return Instrument{}, NewValidationError("owner_id", "owner is required")
	}
	if !input.Criticality.Valid() {
		return Instrument{}, NewValidationError("criticality", "unsupported criticality")
	}
	return Instrument{
		ID: NewID("ins"), LaboratoryID: input.LaboratoryID, AssetNumber: input.AssetNumber,
		Name: input.Name, Model: strings.TrimSpace(input.Model), SerialNumber: strings.TrimSpace(input.SerialNumber),
		Location: strings.TrimSpace(input.Location), OwnerID: input.OwnerID, Criticality: input.Criticality,
		Status: StatusPending, Version: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}, nil
}

func (i *Instrument) ApplyStatus(status InstrumentStatus, reason string, expected Version, now time.Time) error {
	if i.Version != expected {
		return ErrConflict
	}
	if err := RequireTransition(i.Status, status); err != nil {
		return err
	}
	i.Status = status
	i.DisabledReason = ""
	if status == StatusDisabled || status == StatusUnqualified {
		i.DisabledReason = strings.TrimSpace(reason)
		if i.DisabledReason == "" {
			return NewValidationError("reason", "blocking status requires a reason")
		}
	}
	i.Version = i.Version.Next()
	i.UpdatedAt = now.UTC()
	return nil
}

func (i *Instrument) Schedule(nextDue time.Time, expected Version, now time.Time) error {
	if i.Version != expected {
		return ErrConflict
	}
	if !nextDue.After(now) {
		return NewValidationError("next_due_at", "next due date must be in the future")
	}
	i.NextDueAt = nextDue.UTC()
	i.Version = i.Version.Next()
	i.UpdatedAt = now.UTC()
	return nil
}

func (i Instrument) CanBeUsed(now time.Time, warningDays int) bool {
	status := DerivedStatus(i.Status, now, i.NextDueAt, warningDays)
	return status.AllowsUse(i.Criticality == CriticalityCritical)
}
