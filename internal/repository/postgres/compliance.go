package postgres

import (
	"context"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
)

func (s *Store) CreateNonconformance(ctx context.Context, value domain.Nonconformance) error {
	data, err := encode(value)
	if err != nil {
		return err
	}
	_, err = s.exec(ctx, `INSERT INTO nonconformances(id,instrument_id,status,version,data) VALUES($1,$2,$3,$4,$5)`, value.ID, value.InstrumentID, value.Status, value.Version, data)
	return translate(err)
}
func (s *Store) UpdateNonconformance(ctx context.Context, value domain.Nonconformance, expected domain.Version) error {
	data, err := encode(value)
	if err != nil {
		return err
	}
	tag, err := s.exec(ctx, `UPDATE nonconformances SET status=$2,version=$3,data=$4 WHERE id=$1 AND version=$5`, value.ID, value.Status, value.Version, data, expected)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	return nil
}
func (s *Store) GetNonconformance(ctx context.Context, id domain.ID) (domain.Nonconformance, error) {
	var data []byte
	err := s.queryRow(ctx, `SELECT data FROM nonconformances WHERE id=$1`, id).Scan(&data)
	if err != nil {
		return domain.Nonconformance{}, translate(err)
	}
	return decode[domain.Nonconformance](data)
}
func (s *Store) FindOpenByInstrument(ctx context.Context, id domain.ID) (domain.Nonconformance, error) {
	var data []byte
	err := s.queryRow(ctx, `SELECT data FROM nonconformances WHERE instrument_id=$1 AND status<>'closed' ORDER BY created_at DESC LIMIT 1`, id).Scan(&data)
	if err != nil {
		return domain.Nonconformance{}, translate(err)
	}
	return decode[domain.Nonconformance](data)
}
func (s *Store) CreateCertificate(ctx context.Context, value domain.Certificate) error {
	data, err := encode(value)
	if err != nil {
		return err
	}
	_, err = s.exec(ctx, `INSERT INTO certificates(id,number,instrument_id,expires_at,data) VALUES($1,$2,$3,$4,$5)`, value.ID, value.Number, value.InstrumentID, value.ExpiresAt, data)
	return translate(err)
}
func (s *Store) GetCertificate(ctx context.Context, id domain.ID) (domain.Certificate, error) {
	var data []byte
	err := s.queryRow(ctx, `SELECT data FROM certificates WHERE id=$1`, id).Scan(&data)
	if err != nil {
		return domain.Certificate{}, translate(err)
	}
	return decode[domain.Certificate](data)
}
func (s *Store) ListExpiringCertificates(ctx context.Context, from, to time.Time) ([]domain.Certificate, error) {
	rows, err := s.query(ctx, `SELECT data FROM certificates WHERE expires_at BETWEEN $1 AND $2 ORDER BY expires_at`, from, to)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	result := []domain.Certificate{}
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		value, err := decode[domain.Certificate](data)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}
func (s *Store) FindAlertByDeduplication(ctx context.Context, key string) (domain.Alert, error) {
	var data []byte
	err := s.queryRow(ctx, `SELECT data FROM alerts WHERE deduplication_key=$1`, key).Scan(&data)
	if err != nil {
		return domain.Alert{}, translate(err)
	}
	return decode[domain.Alert](data)
}
func (s *Store) RecordAlertScan(ctx context.Context, value domain.Alert, audit domain.AuditEvent) error {
	data, err := encode(value)
	if err != nil {
		return err
	}
	auditData, err := encode(audit)
	if err != nil {
		return err
	}
	return s.WithinTransaction(ctx, func(tx context.Context) error {
		if _, err := s.exec(tx, `INSERT INTO alerts(id,deduplication_key,acknowledged_at,data) VALUES($1,$2,NULL,$3) ON CONFLICT(deduplication_key) DO NOTHING`, value.ID, value.Deduplication, data); err != nil {
			return translate(err)
		}
		_, err := s.exec(tx, `INSERT INTO audit_events(id,entity_type,entity_id,created_at,data) VALUES($1,$2,$3,$4,$5)`, audit.ID, audit.EntityType, audit.EntityID, audit.CreatedAt, auditData)
		return translate(err)
	})
}
func (s *Store) CommitAlertScan(ctx context.Context, value domain.Alert, audit domain.AuditEvent) (domain.Alert, bool, error) {
	var stored domain.Alert
	created := false
	err := s.WithinTransaction(ctx, func(tx context.Context) error {
		data, err := encode(value)
		if err != nil {
			return err
		}
		tag, err := s.exec(tx, `INSERT INTO alerts(id,deduplication_key,acknowledged_at,data) VALUES($1,$2,NULL,$3) ON CONFLICT(deduplication_key) DO NOTHING`, value.ID, value.Deduplication, data)
		if err != nil {
			return translate(err)
		}
		if tag.RowsAffected() == 0 {
			stored, err = s.FindAlertByDeduplication(tx, value.Deduplication)
			return err
		}
		if err := s.AppendAudit(tx, audit); err != nil {
			return err
		}
		stored, created = value, true
		return nil
	})
	return stored, created, err
}
func (s *Store) AcknowledgeAlert(ctx context.Context, value domain.Alert) error {
	data, err := encode(value)
	if err != nil {
		return err
	}
	tag, err := s.exec(ctx, `UPDATE alerts SET acknowledged_at=$2,data=$3 WHERE id=$1`, value.ID, value.AcknowledgedAt, data)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
func (s *Store) ListOpenAlerts(ctx context.Context, request domain.PageRequest) (domain.Page[domain.Alert], error) {
	var total int
	if err := s.queryRow(ctx, `SELECT count(*) FROM alerts WHERE acknowledged_at IS NULL`).Scan(&total); err != nil {
		return domain.Page[domain.Alert]{}, translate(err)
	}
	rows, err := s.query(ctx, `SELECT data FROM alerts WHERE acknowledged_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`, request.Size, (request.Page-1)*request.Size)
	if err != nil {
		return domain.Page[domain.Alert]{}, translate(err)
	}
	defer rows.Close()
	items := []domain.Alert{}
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return domain.Page[domain.Alert]{}, err
		}
		value, err := decode[domain.Alert](data)
		if err != nil {
			return domain.Page[domain.Alert]{}, err
		}
		items = append(items, value)
	}
	return domain.Page[domain.Alert]{Items: items, Page: request.Page, Size: request.Size, Total: total}, rows.Err()
}
