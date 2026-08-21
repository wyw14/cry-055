package domain

import (
	"encoding/json"
	"strings"
	"time"
)

type AuditEvent struct {
	ID         ID              `json:"id"`
	RequestID  string          `json:"request_id"`
	ActorID    ID              `json:"actor_id"`
	Action     string          `json:"action"`
	EntityType string          `json:"entity_type"`
	EntityID   ID              `json:"entity_id"`
	Before     json.RawMessage `json:"before,omitempty"`
	After      json.RawMessage `json:"after,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

func NewAuditEvent(requestID string, actorID ID, action, entityType string, entityID ID, before, after any, now time.Time) (AuditEvent, error) {
	requestID = strings.TrimSpace(requestID)
	action = strings.TrimSpace(action)
	entityType = strings.TrimSpace(entityType)
	if requestID == "" || actorID.Empty() || action == "" || entityType == "" || entityID.Empty() {
		return AuditEvent{}, NewValidationError("audit", "request, actor, action and entity are required")
	}
	beforeJSON, err := marshalAudit(before)
	if err != nil {
		return AuditEvent{}, err
	}
	afterJSON, err := marshalAudit(after)
	if err != nil {
		return AuditEvent{}, err
	}
	return AuditEvent{
		ID: NewID("audit"), RequestID: requestID, ActorID: actorID, Action: action,
		EntityType: entityType, EntityID: entityID, Before: beforeJSON, After: afterJSON,
		CreatedAt: now.UTC(),
	}, nil
}

func marshalAudit(value any) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, NewValidationError("audit_payload", "audit payload is not serializable")
	}
	return encoded, nil
}
