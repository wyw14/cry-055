package postgres

import (
	"context"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

func (s *Store) AppendAudit(ctx context.Context, value domain.AuditEvent) error {
	data, err := encode(value)
	if err != nil {
		return err
	}
	_, err = s.exec(ctx, `INSERT INTO audit_events(id,entity_type,entity_id,created_at,data) VALUES($1,$2,$3,$4,$5)`, value.ID, value.EntityType, value.EntityID, value.CreatedAt, data)
	return translate(err)
}
func (s *Store) ListAudit(ctx context.Context, entityType string, entityID domain.ID) ([]domain.AuditEvent, error) {
	rows, err := s.query(ctx, `SELECT data FROM audit_events WHERE entity_type=$1 AND entity_id=$2 ORDER BY created_at`, entityType, entityID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	result := []domain.AuditEvent{}
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		value, err := decode[domain.AuditEvent](data)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}
func (s *Store) AppendUsageCheck(ctx context.Context, value domain.UsageCheck) error {
	data, err := encode(value)
	if err != nil {
		return err
	}
	_, err = s.exec(ctx, `INSERT INTO usage_checks(id,instrument_id,decision,checked_at,data) VALUES($1,$2,$3,$4,$5)`, value.ID, value.InstrumentID, value.Decision, value.CheckedAt, data)
	return translate(err)
}
func (s *Store) ListUsageChecks(ctx context.Context, instrumentID domain.ID) ([]domain.UsageCheck, error) {
	rows, err := s.query(ctx, `SELECT data FROM usage_checks WHERE instrument_id=$1 ORDER BY checked_at`, instrumentID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	result := []domain.UsageCheck{}
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		value, err := decode[domain.UsageCheck](data)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s *Store) Reserve(ctx context.Context, key, fingerprint string, ttl time.Duration) (bool, error) {
	tag, err := s.exec(ctx, `INSERT INTO idempotency_keys(key,fingerprint,expires_at) VALUES($1,$2,now()+$3::interval) ON CONFLICT(key) DO NOTHING`, key, fingerprint, ttl.String())
	if err != nil {
		return false, translate(err)
	}
	if tag.RowsAffected() == 1 {
		return true, nil
	}
	var existing string
	var expires time.Time
	err = s.queryRow(ctx, `SELECT fingerprint,expires_at FROM idempotency_keys WHERE key=$1`, key).Scan(&existing, &expires)
	if err != nil {
		return false, translate(err)
	}
	if expires.Before(time.Now().UTC()) {
		_, err = s.exec(ctx, `UPDATE idempotency_keys SET fingerprint=$2,completed=false,expires_at=now()+$3::interval WHERE key=$1`, key, fingerprint, ttl.String())
		return true, translate(err)
	}
	if existing != fingerprint {
		return false, domain.ErrConflict
	}
	return false, nil
}
func (s *Store) Complete(ctx context.Context, key, fingerprint string) error {
	tag, err := s.exec(ctx, `UPDATE idempotency_keys SET completed=true WHERE key=$1 AND fingerprint=$2`, key, fingerprint)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	return nil
}
func (s *Store) Release(ctx context.Context, key, fingerprint string) error {
	_, err := s.exec(ctx, `DELETE FROM idempotency_keys WHERE key=$1 AND fingerprint=$2 AND completed=false`, key, fingerprint)
	return translate(err)
}
