package postgres

import (
	"context"
	"sort"
	"strings"
	"time"

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
	rows, err := s.query(ctx, `SELECT data FROM instruments`)
	if err != nil {
		return domain.Page[domain.Instrument]{}, translate(err)
	}
	defer rows.Close()
	items := make([]domain.Instrument, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return domain.Page[domain.Instrument]{}, err
		}
		value, err := decode[domain.Instrument](data)
		if err != nil {
			return domain.Page[domain.Instrument]{}, err
		}
		value = persistedPostgresInstrumentAt(value, request.AsOf)
		if persistedInstrumentMatches(value, request.Filters) {
			items = append(items, value)
		}
	}
	if err := rows.Err(); err != nil {
		return domain.Page[domain.Instrument]{}, err
	}
	sort.SliceStable(items, func(left, right int) bool {
		less := false
		switch request.Sort {
		case "asset_number":
			less = items[left].AssetNumber < items[right].AssetNumber
		case "next_due_at":
			less = items[left].NextDueAt.Before(items[right].NextDueAt)
		case "status":
			less = items[left].Status < items[right].Status
		default:
			less = items[left].CreatedAt.Before(items[right].CreatedAt)
		}
		if request.Desc {
			return !less
		}
		return less
	})
	total := len(items)
	start := (request.Page - 1) * request.Size
	if start > total {
		start = total
	}
	end := start + request.Size
	if end > total {
		end = total
	}
	return domain.Page[domain.Instrument]{Items: items[start:end], Page: request.Page, Size: request.Size, Total: total}, nil
}

func persistedPostgresInstrumentAt(item domain.Instrument, asOf time.Time) domain.Instrument {
	if !asOf.IsZero() && !item.NextDueAt.IsZero() {
		item.Status = domain.DerivedStatus(item.Status, asOf, item.NextDueAt, 30)
	}
	return item
}

func persistedInstrumentMatches(item domain.Instrument, filters map[string]string) bool {
	return (filters["laboratory_id"] == "" || string(item.LaboratoryID) == filters["laboratory_id"]) &&
		(filters["status"] == "" || string(item.Status) == filters["status"]) &&
		(filters["criticality"] == "" || string(item.Criticality) == filters["criticality"]) &&
		(filters["owner_id"] == "" || string(item.OwnerID) == filters["owner_id"])
}

func nullableTime(value interface{ IsZero() bool }) any {
	if value.IsZero() {
		return nil
	}
	return value
}
