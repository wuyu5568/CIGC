package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cigc/app/internal/biz"
	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type dailyCapRepo struct{ data *Data }

// NewDailyCapRepo 当日动态封顶额度与超额冻结。
func NewDailyCapRepo(d *Data) biz.DailyCapRepo { return &dailyCapRepo{data: d} }

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func (r *dailyCapRepo) TakeUnder(ctx context.Context, userID uint64, settleDate time.Time, cap, full decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
	day := dateOnly(settleDate)
	err := r.data.Session(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "settle_date"}},
		DoNothing: true,
	}).Create(&UserDailyDynamicModel{UserID: userID, SettleDate: day, Used: decimal.Zero}).Error
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	var row UserDailyDynamicModel
	if err := r.data.Session(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND settle_date = ?", userID, day).
		First(&row).Error; err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	under, overflow := biz.SplitDailyCap(full, cap, row.Used)
	if under.IsPositive() {
		if err := r.data.Session(ctx).Model(&UserDailyDynamicModel{}).
			Where("user_id = ? AND settle_date = ?", userID, day).
			Update("used", gorm.Expr("used + ?", under)).Error; err != nil {
			return decimal.Zero, decimal.Zero, err
		}
	}
	return under, overflow, nil
}

func (r *dailyCapRepo) AddUsed(ctx context.Context, userID uint64, settleDate time.Time, delta decimal.Decimal) error {
	delta = money.Round(delta)
	if !delta.IsPositive() {
		return nil
	}
	day := dateOnly(settleDate)
	return r.data.Session(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "settle_date"}},
		DoUpdates: clause.Assignments(map[string]any{"used": gorm.Expr("used + ?", delta)}),
	}).Create(&UserDailyDynamicModel{UserID: userID, SettleDate: day, Used: delta}).Error
}

func (r *dailyCapRepo) GetUsed(ctx context.Context, userID uint64, settleDate time.Time) (decimal.Decimal, error) {
	var row UserDailyDynamicModel
	err := r.data.Session(ctx).Where("user_id = ? AND settle_date = ?", userID, dateOnly(settleDate)).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return decimal.Zero, nil
	}
	if err != nil {
		return decimal.Zero, err
	}
	return money.Round(row.Used), nil
}

func (r *dailyCapRepo) CreateHold(ctx context.Context, h *biz.CapOverflowHold) error {
	if h == nil {
		return nil
	}
	m := toHoldModel(h)
	if err := r.data.Session(ctx).Create(m).Error; err != nil {
		return err
	}
	h.ID = m.ID
	return nil
}

func (r *dailyCapRepo) ListActiveHolds(ctx context.Context, userID uint64) ([]*biz.CapOverflowHold, error) {
	var rows []CapOverflowHoldModel
	err := r.data.Session(ctx).
		Where("user_id = ? AND released_at IS NULL AND burned_at IS NULL AND value > 0", userID).
		Order("created_at ASC, id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*biz.CapOverflowHold, len(rows))
	for i := range rows {
		out[i] = toBizHold(&rows[i])
	}
	return out, nil
}

func (r *dailyCapRepo) ActiveTotals(ctx context.Context, userID uint64) (decimal.Decimal, decimal.Decimal, error) {
	var row struct {
		USDT  decimal.Decimal
		Ispay decimal.Decimal
	}
	err := r.data.Session(ctx).Model(&CapOverflowHoldModel{}).
		Select("COALESCE(SUM(usdt),0) AS usdt, COALESCE(SUM(ispay),0) AS ispay").
		Where("user_id = ? AND released_at IS NULL AND burned_at IS NULL", userID).
		Scan(&row).Error
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	return money.Round(row.USDT), money.Round(row.Ispay), nil
}

func (r *dailyCapRepo) ListExpired(ctx context.Context, now time.Time, limit int) ([]*biz.CapOverflowHold, error) {
	if limit <= 0 {
		limit = 200
	}
	var rows []CapOverflowHoldModel
	err := r.data.Session(ctx).
		Where("released_at IS NULL AND burned_at IS NULL AND expires_at IS NOT NULL AND expires_at <= ? AND value > 0", now).
		Order("expires_at ASC, id ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*biz.CapOverflowHold, len(rows))
	for i := range rows {
		out[i] = toBizHold(&rows[i])
	}
	return out, nil
}

func (r *dailyCapRepo) GetHold(ctx context.Context, id uint64) (*biz.CapOverflowHold, error) {
	var row CapOverflowHoldModel
	err := r.data.Session(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toBizHold(&row), nil
}

func (r *dailyCapRepo) SaveHold(ctx context.Context, h *biz.CapOverflowHold) error {
	if h == nil || h.ID == 0 {
		return nil
	}
	return r.data.Session(ctx).Model(&CapOverflowHoldModel{}).Where("id = ?", h.ID).Updates(map[string]any{
		"value":       h.Value,
		"usdt":        h.USDT,
		"ispay":       h.Ispay,
		"expires_at":  h.ExpiresAt,
		"released_at": h.ReleasedAt,
		"burned_at":   h.BurnedAt,
	}).Error
}

func (r *dailyCapRepo) StampUnpackaged(ctx context.Context, beforeSettleDate time.Time, clearHours int) error {
	before := dateOnly(beforeSettleDate)
	hours := biz.ClampOverflowHours(clearHours)
	// 冻结日 0:00 + 24h 封账 + N 小时。
	return r.data.Session(ctx).Model(&CapOverflowHoldModel{}).
		Where("expires_at IS NULL AND released_at IS NULL AND burned_at IS NULL AND value > 0 AND settle_date < ?", before).
		Update("expires_at", gorm.Expr(fmt.Sprintf("DATE_ADD(settle_date, INTERVAL %d HOUR)", 24+hours))).Error
}

func toHoldModel(h *biz.CapOverflowHold) *CapOverflowHoldModel {
	return &CapOverflowHoldModel{
		ID:         h.ID,
		UserID:     h.UserID,
		Value:      h.Value,
		USDT:       h.USDT,
		Ispay:      h.Ispay,
		SourceType: h.SourceType,
		OrderID:    h.OrderID,
		SettleDate: dateOnly(h.SettleDate),
		Remark:     h.Remark,
		CreatedAt:  h.CreatedAt,
		ExpiresAt:  h.ExpiresAt,
		ReleasedAt: h.ReleasedAt,
		BurnedAt:   h.BurnedAt,
	}
}

func toBizHold(m *CapOverflowHoldModel) *biz.CapOverflowHold {
	return &biz.CapOverflowHold{
		ID:         m.ID,
		UserID:     m.UserID,
		Value:      m.Value,
		USDT:       m.USDT,
		Ispay:      m.Ispay,
		SourceType: m.SourceType,
		OrderID:    m.OrderID,
		SettleDate: m.SettleDate,
		Remark:     m.Remark,
		CreatedAt:  m.CreatedAt,
		ExpiresAt:  m.ExpiresAt,
		ReleasedAt: m.ReleasedAt,
		BurnedAt:   m.BurnedAt,
	}
}
