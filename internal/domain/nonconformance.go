package domain

import (
	"strings"
	"time"
)

type NonconformanceStatus string

const (
	NCOpen              NonconformanceStatus = "open"
	NCImpactAssessed    NonconformanceStatus = "impact_assessed"
	NCAwaitingRetest    NonconformanceStatus = "awaiting_retest"
	NCRestorationReview NonconformanceStatus = "restoration_review"
	NCClosed            NonconformanceStatus = "closed"
)

type ImpactedBatch struct {
	BatchNumber string    `json:"batch_number"`
	UsedAt      time.Time `json:"used_at"`
	Product     string    `json:"product"`
	Disposition string    `json:"disposition"`
}

type Nonconformance struct {
	ID              ID                   `json:"id"`
	InstrumentID    ID                   `json:"instrument_id"`
	ExecutionID     ID                   `json:"execution_id"`
	Reason          string               `json:"reason"`
	Status          NonconformanceStatus `json:"status"`
	ImpactedBatches []ImpactedBatch      `json:"impacted_batches"`
	RetestID        ID                   `json:"retest_id,omitempty"`
	RestoreActorID  ID                   `json:"restore_actor_id,omitempty"`
	RestoreComment  string               `json:"restore_comment,omitempty"`
	Version         Version              `json:"version"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

func NewNonconformance(instrumentID, executionID ID, reason string, now time.Time) (Nonconformance, error) {
	if instrumentID.Empty() || executionID.Empty() {
		return Nonconformance{}, NewValidationError("nonconformance", "instrument and execution are required")
	}
	if reason = strings.TrimSpace(reason); reason == "" {
		return Nonconformance{}, NewValidationError("reason", "reason is required")
	}
	return Nonconformance{
		ID: NewID("nc"), InstrumentID: instrumentID, ExecutionID: executionID,
		Reason: reason, Status: NCOpen, Version: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}, nil
}

func (n *Nonconformance) AssessImpact(batches []ImpactedBatch, expected Version, now time.Time) error {
	if n.Version != expected {
		return ErrConflict
	}
	if n.Status != NCOpen {
		return ErrInvalidTransition
	}
	clean := make([]ImpactedBatch, 0, len(batches))
	seen := map[string]struct{}{}
	for _, batch := range batches {
		batch.BatchNumber = strings.TrimSpace(batch.BatchNumber)
		batch.Product = strings.TrimSpace(batch.Product)
		batch.Disposition = strings.TrimSpace(batch.Disposition)
		if batch.BatchNumber == "" || batch.UsedAt.IsZero() {
			return NewValidationError("impacted_batches", "batch number and use time are required")
		}
		if _, ok := seen[batch.BatchNumber]; ok {
			return NewValidationError("impacted_batches", "batch number must be unique")
		}
		seen[batch.BatchNumber] = struct{}{}
		clean = append(clean, batch)
	}
	n.ImpactedBatches = clean
	n.Status = NCImpactAssessed
	n.Version = n.Version.Next()
	n.UpdatedAt = now.UTC()
	return nil
}

func (n *Nonconformance) RequestRetest(expected Version, now time.Time) error {
	if n.Version != expected {
		return ErrConflict
	}
	if n.Status != NCImpactAssessed {
		return ErrInvalidTransition
	}
	n.Status = NCAwaitingRetest
	n.Version = n.Version.Next()
	n.UpdatedAt = now.UTC()
	return nil
}

func (n *Nonconformance) AttachRetest(retestID ID, qualified bool, expected Version, now time.Time) error {
	if n.Version != expected {
		return ErrConflict
	}
	if n.Status != NCAwaitingRetest || retestID.Empty() {
		return ErrInvalidTransition
	}
	n.RetestID = retestID
	if qualified {
		n.Status = NCRestorationReview
	} else {
		n.Status = NCAwaitingRetest
	}
	n.Version = n.Version.Next()
	n.UpdatedAt = now.UTC()
	return nil
}

func (n *Nonconformance) ConfirmRestoration(actor Actor, comment string, expected Version, now time.Time) error {
	if n.Version != expected {
		return ErrConflict
	}
	if n.Status != NCRestorationReview || !actor.HasRole("quality_manager") {
		return ErrUnauthorized
	}
	if comment = strings.TrimSpace(comment); comment == "" {
		return NewValidationError("comment", "restoration comment is required")
	}
	n.RestoreActorID = actor.ID
	n.RestoreComment = comment
	n.Status = NCClosed
	n.Version = n.Version.Next()
	n.UpdatedAt = now.UTC()
	return nil
}
