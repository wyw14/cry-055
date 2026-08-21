package domain

import (
	"math"
	"strings"
	"time"
)

type CalibrationConclusion string

const (
	ConclusionQualified   CalibrationConclusion = "qualified"
	ConclusionUnqualified CalibrationConclusion = "unqualified"
)

type Measurement struct {
	Point    string  `json:"point"`
	Expected float64 `json:"expected"`
	Actual   float64 `json:"actual"`
	Error    float64 `json:"error"`
	Passed   bool    `json:"passed"`
}

func NewMeasurement(point string, expected, actual, tolerance float64) (Measurement, error) {
	point = strings.TrimSpace(point)
	if point == "" || tolerance <= 0 {
		return Measurement{}, NewValidationError("measurement", "point and tolerance are required")
	}
	delta := math.Abs(actual - expected)
	return Measurement{Point: point, Expected: expected, Actual: actual, Error: delta, Passed: delta <= tolerance}, nil
}

type CalibrationExecution struct {
	ID            ID                    `json:"id"`
	RootID        ID                    `json:"root_id"`
	InstrumentID  ID                    `json:"instrument_id"`
	ItemID        ID                    `json:"item_id"`
	PlanID        ID                    `json:"plan_id"`
	ExecutorID    ID                    `json:"executor_id"`
	Measurements  []Measurement         `json:"measurements"`
	Conclusion    CalibrationConclusion `json:"conclusion"`
	CertificateID ID                    `json:"certificate_id"`
	CompletedAt   time.Time             `json:"completed_at"`
	ReviewerID    ID                    `json:"reviewer_id"`
	ReviewedAt    *time.Time            `json:"reviewed_at,omitempty"`
	ReviewComment string                `json:"review_comment,omitempty"`
	Version       Version               `json:"version"`
	PreviousID    ID                    `json:"previous_id,omitempty"`
	CreatedAt     time.Time             `json:"created_at"`
}

func NewExecution(instrumentID, itemID, planID, executorID ID, measurements []Measurement, certificateID ID, completedAt, now time.Time) (CalibrationExecution, error) {
	if instrumentID.Empty() || itemID.Empty() || executorID.Empty() {
		return CalibrationExecution{}, NewValidationError("execution", "instrument, item and executor are required")
	}
	if len(measurements) == 0 {
		return CalibrationExecution{}, NewValidationError("measurements", "at least one measurement is required")
	}
	conclusion := ConclusionQualified
	for _, measurement := range measurements {
		if !measurement.Passed {
			conclusion = ConclusionUnqualified
			break
		}
	}
	id := NewID("exec")
	return CalibrationExecution{
		ID: id, RootID: id, InstrumentID: instrumentID, ItemID: itemID, PlanID: planID,
		ExecutorID: executorID, Measurements: append([]Measurement(nil), measurements...),
		Conclusion: conclusion, CertificateID: certificateID, CompletedAt: completedAt.UTC(),
		Version: 1, CreatedAt: now.UTC(),
	}, nil
}

func (e *CalibrationExecution) Review(reviewer ID, comment string, now time.Time) error {
	if reviewer.Empty() || reviewer == e.ExecutorID {
		return NewValidationError("reviewer_id", "independent reviewer is required")
	}
	if e.ReviewedAt != nil {
		return ErrConflict
	}
	stamp := now.UTC()
	e.ReviewerID = reviewer
	e.ReviewedAt = &stamp
	e.ReviewComment = strings.TrimSpace(comment)
	e.Version = e.Version.Next()
	return nil
}

func (e CalibrationExecution) Revise(measurements []Measurement, actor ID, now time.Time) (CalibrationExecution, error) {
	revised, err := NewExecution(e.InstrumentID, e.ItemID, e.PlanID, actor, measurements, e.CertificateID, e.CompletedAt, now)
	if err != nil {
		return CalibrationExecution{}, err
	}
	revised.RootID = e.ID
	revised.PreviousID = e.ID
	return revised, nil
}
