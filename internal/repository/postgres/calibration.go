package postgres

import (
	"context"

	"github.com/wyw14/cry-055/internal/domain"
)

func (s *Store) CreateItem(ctx context.Context, value domain.CalibrationItem) error {
	data, err := encode(value)
	if err != nil {
		return err
	}
	_, err = s.exec(ctx, `INSERT INTO calibration_items(id,code,version,data) VALUES($1,$2,$3,$4)`, value.ID, value.Code, value.Version, data)
	return translate(err)
}
func (s *Store) GetItem(ctx context.Context, id domain.ID) (domain.CalibrationItem, error) {
	var data []byte
	err := s.queryRow(ctx, `SELECT data FROM calibration_items WHERE id=$1`, id).Scan(&data)
	if err != nil {
		return domain.CalibrationItem{}, translate(err)
	}
	return decode[domain.CalibrationItem](data)
}
func (s *Store) CreatePlan(ctx context.Context, value domain.CalibrationPlan) error {
	data, err := encode(value)
	if err != nil {
		return err
	}
	_, err = s.exec(ctx, `INSERT INTO calibration_plans(id,instrument_id,item_id,status,due_at,version,data) VALUES($1,$2,$3,$4,$5,$6,$7)`, value.ID, value.InstrumentID, value.ItemID, value.Status, value.DueAt, value.Version, data)
	return translate(err)
}
func (s *Store) UpdatePlan(ctx context.Context, value domain.CalibrationPlan, expected domain.Version) error {
	data, err := encode(value)
	if err != nil {
		return err
	}
	tag, err := s.exec(ctx, `UPDATE calibration_plans SET status=$2,due_at=$3,version=$4,data=$5 WHERE id=$1 AND version=$6`, value.ID, value.Status, value.DueAt, value.Version, data, expected)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	return nil
}
func (s *Store) GetPlan(ctx context.Context, id domain.ID) (domain.CalibrationPlan, error) {
	var data []byte
	err := s.queryRow(ctx, `SELECT data FROM calibration_plans WHERE id=$1`, id).Scan(&data)
	if err != nil {
		return domain.CalibrationPlan{}, translate(err)
	}
	return decode[domain.CalibrationPlan](data)
}
func (s *Store) FindOpenPlan(ctx context.Context, instrumentID, itemID domain.ID) (domain.CalibrationPlan, error) {
	var data []byte
	err := s.queryRow(ctx, `SELECT data FROM calibration_plans WHERE instrument_id=$1 AND item_id=$2 AND status NOT IN ('completed','cancelled') ORDER BY created_at DESC LIMIT 1`, instrumentID, itemID).Scan(&data)
	if err != nil {
		return domain.CalibrationPlan{}, translate(err)
	}
	return decode[domain.CalibrationPlan](data)
}
func (s *Store) CreateExecution(ctx context.Context, value domain.CalibrationExecution) error {
	root := value.RootID
	if root.Empty() {
		root = value.ID
	}
	value.RootID = root
	return s.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.StageExecutionRevision(tx, value); err != nil {
			return err
		}
		_, err := s.exec(tx, `INSERT INTO calibration_execution_heads(root_id,head_id) VALUES($1,$2)`, root, value.ID)
		return translate(err)
	})
}
func (s *Store) StageExecutionRevision(ctx context.Context, value domain.CalibrationExecution) error {
	data, err := encode(value)
	if err != nil {
		return err
	}
	_, err = s.exec(ctx, `INSERT INTO calibration_executions(id,root_id,instrument_id,item_id,conclusion,version,data) VALUES($1,$2,$3,$4,$5,$6,$7)`, value.ID, value.RootID, value.InstrumentID, value.ItemID, value.Conclusion, value.Version, data)
	return translate(err)
}
func (s *Store) LinkExecutionRevision(ctx context.Context, value domain.CalibrationExecution) error {
	_, err := s.exec(ctx, `INSERT INTO calibration_execution_heads(root_id,head_id) VALUES($1,$2) ON CONFLICT(root_id) DO UPDATE SET head_id=EXCLUDED.head_id`, value.RootID, value.ID)
	return translate(err)
}
func (s *Store) GetExecution(ctx context.Context, id domain.ID) (domain.CalibrationExecution, error) {
	var data []byte
	err := s.queryRow(ctx, `SELECT data FROM calibration_executions WHERE id=$1`, id).Scan(&data)
	if err != nil {
		return domain.CalibrationExecution{}, translate(err)
	}
	return decode[domain.CalibrationExecution](data)
}
func (s *Store) ListExecutionVersions(ctx context.Context, id domain.ID) ([]domain.CalibrationExecution, error) {
	rows, err := s.query(ctx, `SELECT data FROM calibration_executions WHERE root_id=COALESCE((SELECT root_id FROM calibration_executions WHERE id=$1),$1) ORDER BY version,created_at`, id)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	result := []domain.CalibrationExecution{}
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		value, err := decode[domain.CalibrationExecution](data)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil, domain.ErrNotFound
	}
	return result, rows.Err()
}
