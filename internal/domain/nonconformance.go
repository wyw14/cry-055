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

type RestorationEvidence struct {
	CaseVersion        Version
	ImpactRecorded     bool
	RetestRequested    bool
	QualifiedRetestID  ID
	RestorationPending bool
}

type RestorationAuthorization struct {
	CaseVersion Version
	ActorID     ID
	RetestID    ID
	Comment     string
}

func (n Nonconformance) CollectRestorationEvidence() RestorationEvidence {
	evidence := RestorationEvidence{
		CaseVersion:       n.Version,
		ImpactRecorded:    len(n.ImpactedBatches) > 0,
		QualifiedRetestID: n.RetestID,
	}
	switch n.Status {
	case NCAwaitingRetest:
		evidence.RetestRequested = true
	case NCRestorationReview:
		evidence.RetestRequested = true
		evidence.RestorationPending = true
	}
	return evidence
}

func (e RestorationEvidence) Authorize(actor Actor, comment string, expected Version) (RestorationAuthorization, error) {
	if expected != e.CaseVersion {
		return RestorationAuthorization{}, ErrConflict
	}
	if actor.ID.Empty() || !actor.HasRole("quality_manager") {
		return RestorationAuthorization{}, ErrUnauthorized
	}
	comment = strings.TrimSpace(comment)
	if comment == "" {
		return RestorationAuthorization{}, NewValidationError("comment", "restoration comment is required")
	}
	// Restoration is authorized only once an independent, qualified retest has
	// been attached and persisted (status restoration_review). RetestRequested
	// alone is insufficient: it marks that a retest was asked for, not that the
	// reviewed qualified evidence landed in the store. Authorizing before the
	// evidence is attached lets a concurrent restore close the case and the
	// pending retest then loses to a version conflict, so require RestorationPending.
	if !e.ImpactRecorded || !e.RestorationPending {
		return RestorationAuthorization{}, ErrUnauthorized
	}
	return RestorationAuthorization{
		CaseVersion: e.CaseVersion,
		ActorID:     actor.ID,
		RetestID:    e.QualifiedRetestID,
		Comment:     comment,
	}, nil
}

func (n *Nonconformance) ApplyRestoration(authorization RestorationAuthorization, now time.Time) error {
	if n.Version != authorization.CaseVersion {
		return ErrConflict
	}
	// Restoration closes the case only after independent, qualified retest
	// evidence has been attached and persisted (restoration_review). awaiting_retest
	// means a retest was requested but no reviewed qualified evidence landed yet,
	// so closing then is not permitted — a concurrent attach would lose to a
	// version conflict and the evidence would be silently dropped.
	if n.Status != NCRestorationReview || n.RetestID != authorization.RetestID || n.RetestID.Empty() {
		return ErrInvalidTransition
	}
	n.RestoreActorID = authorization.ActorID
	n.RestoreComment = authorization.Comment
	n.Status = NCClosed
	n.Version = n.Version.Next()
	n.UpdatedAt = now.UTC()
	return nil
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
	evidence := n.CollectRestorationEvidence()
	authorization, err := evidence.Authorize(actor, comment, expected)
	if err != nil {
		return err
	}
	return n.ApplyRestoration(authorization, now)
}
