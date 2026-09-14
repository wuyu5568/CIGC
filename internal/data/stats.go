package data

import (
	"context"
	"time"

	"github.com/cigc/app/internal/biz"
	"github.com/shopspring/decimal"
)

type statsRepo struct{ data *Data }

// NewStatsRepo 管理端汇总仓储。
func NewStatsRepo(d *Data) biz.StatsRepo { return &statsRepo{data: d} }

func (r *statsRepo) CountUsers(ctx context.Context) (total, activated int64, err error) {
	if err = r.data.Session(ctx).Model(&UserModel{}).Count(&total).Error; err != nil {
		return
	}
	err = r.data.Session(ctx).Model(&UserModel{}).
		Where("paid_amount > 0").Count(&activated).Error
	return
}

func (r *statsRepo) CountUsersCreatedBetween(ctx context.Context, from, to time.Time) (int64, error) {
	var n int64
	err := r.data.Session(ctx).Model(&UserModel{}).
		Where("created_at >= ? AND created_at < ?", from, to).
		Count(&n).Error
	return n, err
}

func (r *statsRepo) CountFirstActivatedBetween(ctx context.Context, from, to time.Time) (int64, error) {
	var n int64
	err := r.data.Session(ctx).Raw(`
SELECT COUNT(*) FROM (
  SELECT user_id, MIN(paid_at) AS first_paid
  FROM orders
  WHERE status = ? AND paid_at IS NOT NULL
  GROUP BY user_id
) t WHERE first_paid >= ? AND first_paid < ?`, biz.OrderPaid, from, to).Scan(&n).Error
	return n, err
}

func (r *statsRepo) SumPaidOrders(ctx context.Context, from, to *time.Time) (decimal.Decimal, error) {
	q := r.data.Session(ctx).Model(&OrderModel{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("status = ?", biz.OrderPaid)
	if from != nil && to != nil {
		q = q.Where("paid_at >= ? AND paid_at < ?", *from, *to)
	}
	var sum decimal.Decimal
	err := q.Scan(&sum).Error
	return sum, err
}

func (r *statsRepo) SumAvailableUSDT(ctx context.Context) (decimal.Decimal, error) {
	var sum decimal.Decimal
	err := r.data.Session(ctx).Model(&UserModel{}).
		Select("COALESCE(SUM(available_balance), 0)").
		Scan(&sum).Error
	return sum, err
}

func (r *statsRepo) SumIspay(ctx context.Context) (decimal.Decimal, error) {
	var sum decimal.Decimal
	err := r.data.Session(ctx).Model(&UserModel{}).
		Select("COALESCE(SUM(ispay_balance + frozen_ispay), 0)").
		Scan(&sum).Error
	return sum, err
}

func (r *statsRepo) SumLedgerUSDT(ctx context.Context, types []string, from, to *time.Time) (decimal.Decimal, error) {
	if len(types) == 0 {
		return decimal.Zero, nil
	}
	q := r.data.Session(ctx).Model(&LedgerEntryModel{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("entry_type IN ? AND amount > 0", types).
		Where("balance_kind IN ?", []string{biz.BalanceAvailable, biz.BalanceLock})
	if from != nil && to != nil {
		q = q.Where("created_at >= ? AND created_at < ?", *from, *to)
	}
	var sum decimal.Decimal
	err := q.Scan(&sum).Error
	return sum, err
}

func (r *statsRepo) SumWithdrawUSDT(ctx context.Context, from, to *time.Time) (decimal.Decimal, error) {
	q := r.data.Session(ctx).Model(&WithdrawModel{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("status IN ?", []string{biz.WithdrawPending, biz.WithdrawRewarded, biz.WithdrawDoing, biz.WithdrawPass}).
		Where("(asset = ? OR asset = '' OR asset IS NULL)", biz.WithdrawAssetUSDT)
	if from != nil && to != nil {
		q = q.Where("created_at >= ? AND created_at < ?", *from, *to)
	}
	var sum decimal.Decimal
	err := q.Scan(&sum).Error
	return sum, err
}
