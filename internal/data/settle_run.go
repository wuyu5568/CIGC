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
		StaticCount: row.StaticCount,
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
	return strings.Contains(msg, "Duplicate") || strings.Contains(msg, "UNIQUE") || strings.Contains(msg, "1062")
}

func isForeignKey(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && (mysqlErr.Number == 1451 || mysqlErr.Number == 1452) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "foreign key") || strings.Contains(msg, "1451")
}

func isMissingTable(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1146 {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "Error 1146") || strings.Contains(msg, "1146 (42S02)")
}

func (r *settleRunRepo) DeleteAfter(ctx context.Context, after time.Time) (int, error) {
	day := after.Format("2006-01-02")
	res := r.data.db.WithContext(ctx).Where("settle_date > ?", day).Delete(&SettleRunModel{})
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
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
		StaticCount: run.StaticCount,
		Remark:      run.Remark,
	}
	return r.data.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "settle_date"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"forced", "user_count", "cap_updated", "direct_count",
			"match_count", "manage_count", "static_count", "remark", "updated_at",
		}),
	}).Create(&row).Error
}
