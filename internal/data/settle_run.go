package data

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cigc/app/internal/biz"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type settleRunRepo struct{ data *Data }

// NewSettleRunRepo 日结占位仓储。
func NewSettleRunRepo(d *Data) biz.SettleRunRepo { return &settleRunRepo{data: d} }

func toSettleRun(row SettleRunModel) *biz.SettleRun {
	return &biz.SettleRun{
		ID:          row.ID,
		SettleDate:  row.SettleDate,
		Forced:      row.Forced,
		UserCount:   row.UserCount,
		CapUpdated:  row.CapUpdated,
		DirectCount: row.DirectCount,
		MatchCount:  row.MatchCount,
		ManageCount: row.ManageCount,
		Remark:      row.Remark,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func (r *settleRunRepo) FindByDate(ctx context.Context, settleDate time.Time) (*biz.SettleRun, error) {
	day := settleDate.Format("2006-01-02")
	var row SettleRunModel
	err := r.data.db.WithContext(ctx).Where("settle_date = ?", day).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toSettleRun(row), nil
}

func (r *settleRunRepo) FindLatest(ctx context.Context) (*biz.SettleRun, error) {
	var row SettleRunModel
	err := r.data.db.WithContext(ctx).Order("settle_date DESC").First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toSettleRun(row), nil
}

func (r *settleRunRepo) TryClaim(ctx context.Context, settleDate time.Time, remark string) (bool, error) {
	day := time.Date(settleDate.Year(), settleDate.Month(), settleDate.Day(), 0, 0, 0, 0, time.UTC)
	row := SettleRunModel{
		SettleDate: day,
		Forced:     false,
		Remark:     remark,
	}
	err := r.data.db.WithContext(ctx).Create(&row).Error
	if err != nil {
		if isDuplicateKey(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") || strings.Contains(msg, "1062")
}

func (r *settleRunRepo) Upsert(ctx context.Context, run *biz.SettleRun) error {
	day := time.Date(run.SettleDate.Year(), run.SettleDate.Month(), run.SettleDate.Day(), 0, 0, 0, 0, time.UTC)
	row := SettleRunModel{
		SettleDate:  day,
		Forced:      run.Forced,
		UserCount:   run.UserCount,
		CapUpdated:  run.CapUpdated,
		DirectCount: run.DirectCount,
		MatchCount:  run.MatchCount,
		ManageCount: run.ManageCount,
		Remark:      run.Remark,
	}
	return r.data.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "settle_date"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"forced", "user_count", "cap_updated", "direct_count",
			"match_count", "manage_count", "remark", "updated_at",
		}),
	}).Create(&row).Error
}
