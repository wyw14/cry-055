package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/wyw14/cry-055/internal/domain"
)

func (s *Store) CreateLaboratory(ctx context.Context, value domain.Laboratory) error {
	data, err := encode(value)
	if err != nil {
		return err
	}
	_, err = s.exec(ctx, `INSERT INTO laboratories(id, code, version, data) VALUES($1,$2,$3,$4)`, value.ID, value.Code, value.Version, data)
	return translate(err)
}

func (s *Store) GetLaboratory(ctx context.Context, id domain.ID) (domain.Laboratory, error) {
	var data []byte
	err := s.queryRow(ctx, `SELECT data FROM laboratories WHERE id=$1`, id).Scan(&data)
	if err != nil {
		return domain.Laboratory{}, translate(err)
	}
	return decode[domain.Laboratory](data)
}

func (s *Store) CreateInstrument(ctx context.Context, value domain.Instrument) error {
	data, err := encode(value)
	if err != nil {
		return err
	}
	_, err = s.exec(ctx, `INSERT INTO instruments(id,asset_number,laboratory_id,status,criticality,next_due_at,version,data) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		value.ID, value.AssetNumber, value.LaboratoryID, value.Status, value.Criticality, nullableTime(value.NextDueAt), value.Version, data)
	return translate(err)
}

func (s *Store) UpdateInstrument(ctx context.Context, value domain.Instrument, expected domain.Version) error {
	data, err := encode(value)
	if err != nil {
		return err
	}
	tag, err := s.exec(ctx, `UPDATE instruments SET asset_number=$2,laboratory_id=$3,status=$4,criticality=$5,next_due_at=$6,version=$7,data=$8 WHERE id=$1 AND version=$9`,
		value.ID, value.AssetNumber, value.LaboratoryID, value.Status, value.Criticality, nullableTime(value.NextDueAt), value.Version, data, expected)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	return nil
}

func (s *Store) GetInstrument(ctx context.Context, id domain.ID) (domain.Instrument, error) {
	var data []byte
	err := s.queryRow(ctx, `SELECT data FROM instruments WHERE id=$1`, id).Scan(&data)
	if err != nil {
		return domain.Instrument{}, translate(err)
	}
	return decode[domain.Instrument](data)
}

func (s *Store) GetInstrumentByAssetNumber(ctx context.Context, asset string) (domain.Instrument, error) {
	var data []byte
	err := s.queryRow(ctx, `SELECT data FROM instruments WHERE asset_number=$1`, strings.ToUpper(strings.TrimSpace(asset))).Scan(&data)
	if err != nil {
		return domain.Instrument{}, translate(err)
	}
	return decode[domain.Instrument](data)
}

func (s *Store) ListInstruments(ctx context.Context, request domain.PageRequest) (domain.Page[domain.Instrument], error) {
	where, args := instrumentWhere(request.Filters)
	var total int
	if err := s.queryRow(ctx, `SELECT count(*) FROM instruments`+where, args...).Scan(&total); err != nil {
		return domain.Page[domain.Instrument]{}, translate(err)
	}
	columns := map[string]string{"created_at": "created_at", "asset_number": "asset_number", "next_due_at": "next_due_at", "status": "status"}
	direction := "ASC"
	if request.Desc {
		direction = "DESC"
	}
	args = append(args, request.Size, (request.Page-1)*request.Size)
	query := fmt.Sprintf(`SELECT data FROM instruments%s ORDER BY %s %s NULLS LAST LIMIT $%d OFFSET $%d`, where, columns[request.Sort], direction, len(args)-1, len(args))
	rows, err := s.query(ctx, query, args...)
	if err != nil {
		return domain.Page[domain.Instrument]{}, translate(err)
	}
	defer rows.Close()
	items := make([]domain.Instrument, 0, request.Size)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return domain.Page[domain.Instrument]{}, err
		}
		value, err := decode[domain.Instrument](data)
		if err != nil {
			return domain.Page[domain.Instrument]{}, err
		}
		items = append(items, value)
	}
	return domain.Page[domain.Instrument]{Items: items, Page: request.Page, Size: request.Size, Total: total}, rows.Err()
}

func instrumentWhere(filters map[string]string) (string, []any) {
	parts := make([]string, 0, len(filters))
	args := make([]any, 0, len(filters))
	columns := map[string]string{"laboratory_id": "laboratory_id", "status": "status", "criticality": "criticality", "owner_id": "data->>'owner_id'"}
	for _, key := range []string{"laboratory_id", "status", "criticality", "owner_id"} {
		if value := filters[key]; value != "" {
			args = append(args, value)
			parts = append(parts, fmt.Sprintf("%s=$%d", columns[key], len(args)))
		}
	}
	if len(parts) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(parts, " AND "), args
}

func nullableTime(value interface{ IsZero() bool }) any {
	if value.IsZero() {
		return nil
	}
	return value
}
