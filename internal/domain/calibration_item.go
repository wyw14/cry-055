package domain

import (
	"strings"
	"time"
)

type CalibrationItem struct {
	ID                  ID        `json:"id"`
	Code                string    `json:"code"`
	Name                string    `json:"name"`
	PeriodDays          int       `json:"period_days"`
	WarningDays         int       `json:"warning_days"`
	Tolerance           float64   `json:"tolerance"`
	Unit                string    `json:"unit"`
	ReferenceStandardID ID        `json:"reference_standard_id"`
	ApplicableModels    []string  `json:"applicable_models"`
	Version             Version   `json:"version"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func NewCalibrationItem(code, name string, periodDays, warningDays int, tolerance float64, unit string, standard ID, models []string, now time.Time) (CalibrationItem, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	name = strings.TrimSpace(name)
	if code == "" || name == "" {
		return CalibrationItem{}, NewValidationError("code", "code and name are required")
	}
	if periodDays <= 0 || warningDays < 0 || warningDays >= periodDays {
		return CalibrationItem{}, NewValidationError("period_days", "period and warning window are invalid")
	}
	if tolerance <= 0 || strings.TrimSpace(unit) == "" {
		return CalibrationItem{}, NewValidationError("tolerance", "positive tolerance and unit are required")
	}
	if standard.Empty() {
		return CalibrationItem{}, NewValidationError("reference_standard_id", "reference standard is required")
	}
	cleanModels := make([]string, 0, len(models))
	seen := map[string]struct{}{}
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if _, exists := seen[model]; !exists {
			seen[model] = struct{}{}
			cleanModels = append(cleanModels, model)
		}
	}
	return CalibrationItem{
		ID: NewID("item"), Code: code, Name: name, PeriodDays: periodDays,
		WarningDays: warningDays, Tolerance: tolerance, Unit: strings.TrimSpace(unit),
		ReferenceStandardID: standard, ApplicableModels: cleanModels,
		Version: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}, nil
}

func (c CalibrationItem) NextDue(completedAt time.Time) time.Time {
	return completedAt.UTC().AddDate(0, 0, c.PeriodDays)
}

func (c CalibrationItem) AppliesTo(model string) bool {
	if len(c.ApplicableModels) == 0 {
		return true
	}
	for _, candidate := range c.ApplicableModels {
		if candidate == model {
			return true
		}
	}
	return false
}
