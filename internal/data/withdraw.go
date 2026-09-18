package data

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cigc/app/internal/biz"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type withdrawRepo struct{ data *Data }

// NewWithdrawRepo 提现仓储。
func NewWithdrawRepo(d *Data) biz.WithdrawRepo { return &withdrawRepo{data: d} }

func toBizWithdraw(m *WithdrawModel) *biz.Withdraw {
	return &biz.Withdraw{
		ID:             m.ID,
		UserID:         m.UserID,
		Amount:         m.Amount,
		FeeAmount:      m.FeeAmount,
		CreditedAmount: m.CreditedAmount,
		Asset:          m.Asset,
		Status:         m.Status,
		Remark:         m.Remark,
		TxHash:         m.TxHash,
		PayoutError:    m.PayoutError,
		ReviewedAt:     m.ReviewedAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func (r *withdrawRepo) Create(ctx context.Context, w *biz.Withdraw) (*biz.Withdraw, error) {
	m := WithdrawModel{
		UserID:         w.UserID,
		Amount:         w.Amount,
		FeeAmount:      w.FeeAmount,
		CreditedAmount: w.CreditedAmount,
		Asset:          w.Asset,
		Status:         w.Status,
		Remark:         w.Remark,
		TxHash:         w.TxHash,
		PayoutError:    w.PayoutError,
		CreatedAt:      w.CreatedAt,
	}
	if err := r.data.Session(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	return r.FindByID(ctx, m.ID)
}

func (r *withdrawRepo) FindByID(ctx context.Context, id uint64) (*biz.Withdraw, error) {
	var m WithdrawModel
	if err := r.data.Session(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrWithdrawNotFound
		}
		return nil, err
	}
	return toBizWithdraw(&m), nil
}

func (r *withdrawRepo) CasStatus(ctx context.Context, id uint64, from, to, remark string, reviewedAt *time.Time) error {
	res := r.data.Session(ctx).Model(&WithdrawModel{}).
		Where("id = ? AND status = ?", id, from).
		Updates(map[string]any{
			"status":      to,
			"remark":      remark,
			"reviewed_at": reviewedAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return biz.ErrWithdrawConflict
	}
	return nil
}

func (r *withdrawRepo) ListByUser(ctx context.Context, userID uint64) ([]*biz.Withdraw, error) {
	var rows []WithdrawModel
	if err := r.data.Session(ctx).Where("user_id = ?", userID).
		Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.Withdraw, len(rows))
	for i := range rows {
		out[i] = toBizWithdraw(&rows[i])
	}
	return out, nil
}

func (r *withdrawRepo) ListAdmin(ctx context.Context, address, status, asset string, page, pageSize int) ([]*biz.AdminWithdrawRow, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = biz.DefaultRewardPageSize
	}
	base := r.data.Session(ctx).Table("withdraws").
		Joins("LEFT JOIN users ON users.id = withdraws.user_id")
	if address != "" {
		base = base.Where("users.address LIKE ?", "%"+address+"%")
	}
	if status != "" {
		base = base.Where("withdraws.status = ?", status)
	}
	if asset != "" {
		base = base.Where("withdraws.asset = ?", asset)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type row struct {
		WithdrawModel
		Address string `gorm:"column:address"`
	}
	var rows []row
	offset := (page - 1) * pageSize
	if err := base.Session(&gorm.Session{}).
		Select("withdraws.*, users.address AS address").
		Order("withdraws.id DESC").
		Offset(offset).Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminWithdrawRow, len(rows))
	for i := range rows {
		w := toBizWithdraw(&rows[i].WithdrawModel)
		out[i] = &biz.AdminWithdrawRow{Withdraw: *w, Address: rows[i].Address}
	}
	return out, int(total), nil
}

func (r *withdrawRepo) ListPayoutQueue(ctx context.Context, limit int) ([]*biz.AdminWithdrawRow, error) {
	if limit < 1 {
		limit = 20
	}
	type row struct {
		WithdrawModel
		Address string `gorm:"column:address"`
	}
	var rows []row
	err := r.data.Session(ctx).Table("withdraws").
		Joins("LEFT JOIN users ON users.id = withdraws.user_id").
		Where("withdraws.status IN ?", []string{biz.WithdrawRewarded, biz.WithdrawDoing}).
		Where("(withdraws.asset = ? OR withdraws.asset = ? OR withdraws.asset = '' OR withdraws.asset IS NULL)", biz.WithdrawAssetUSDT, biz.WithdrawAssetIspay).
		Select("withdraws.*, users.address AS address").
		Order("withdraws.id ASC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*biz.AdminWithdrawRow, len(rows))
	for i := range rows {
		w := toBizWithdraw(&rows[i].WithdrawModel)
		out[i] = &biz.AdminWithdrawRow{Withdraw: *w, Address: rows[i].Address}
	}
	return out, nil
}

func (r *withdrawRepo) SumUsedToday(ctx context.Context, userID uint64, asset string, from, to time.Time) (decimal.Decimal, error) {
	if userID == 0 {
		return decimal.Zero, nil
	}
	var sum decimal.Decimal
	q := r.data.Session(ctx).Model(&WithdrawModel{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, from, to).
		Where("status IN ?", []string{biz.WithdrawPending, biz.WithdrawRewarded, biz.WithdrawDoing, biz.WithdrawPass})
	if asset == biz.WithdrawAssetUSDT {
		q = q.Where("(asset = ? OR asset = '' OR asset IS NULL)", biz.WithdrawAssetUSDT)
	} else if asset != "" {
		q = q.Where("asset = ?", asset)
	}
	if err := q.Scan(&sum).Error; err != nil {
		return decimal.Zero, err
	}
	return sum, nil
}

func (r *withdrawRepo) UpdatePayoutMeta(ctx context.Context, id uint64, txHash, payoutError string) error {
	res := r.data.Session(ctx).Model(&WithdrawModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"tx_hash":      txHash,
			"payout_error": payoutError,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return biz.ErrWithdrawNotFound
	}
	return nil
}

type configRepo struct{ data *Data }

// NewConfigRepo 业务配置仓储。
func NewConfigRepo(d *Data) biz.ConfigRepo { return &configRepo{data: d} }

func (r *configRepo) GetValue(ctx context.Context, key string) (string, error) {
	var m BusinessConfigModel
	err := r.data.Session(ctx).Where("config_key = ?", key).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return m.Value, nil
}

func toBizConfig(m *BusinessConfigModel) *biz.BusinessConfig {
	return &biz.BusinessConfig{
		ID:        m.ID,
		Key:       m.ConfigKey,
		Name:      m.Name,
		Value:     m.Value,
		SortOrder: m.SortOrder,
	}
}

func (r *configRepo) List(ctx context.Context) ([]*biz.BusinessConfig, error) {
	var rows []BusinessConfigModel
	if err := r.data.Session(ctx).Order("sort_order ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.BusinessConfig, len(rows))
	for i := range rows {
		out[i] = toBizConfig(&rows[i])
	}
	return out, nil
}

func (r *configRepo) FindByID(ctx context.Context, id uint64) (*biz.BusinessConfig, error) {
	var m BusinessConfigModel
	err := r.data.Session(ctx).First(&m, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toBizConfig(&m), nil
}

func (r *configRepo) SetValue(ctx context.Context, id uint64, value string) error {
	res := r.data.Session(ctx).Model(&BusinessConfigModel{}).
		Where("id = ?", id).
		Update("value", value)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return biz.ErrConfigNotFound
	}
	return nil
}

func (r *configRepo) Upsert(ctx context.Context, row *biz.BusinessConfig) error {
	if row == nil || strings.TrimSpace(row.Key) == "" {
		return biz.ErrConfigInvalid
	}
	var m BusinessConfigModel
	err := r.data.Session(ctx).Where("config_key = ?", row.Key).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		m = BusinessConfigModel{
			ConfigKey: row.Key,
			Name:      row.Name,
			Value:     row.Value,
			SortOrder: row.SortOrder,
		}
		return r.data.Session(ctx).Create(&m).Error
	}
	if err != nil {
		return err
	}
	return r.data.Session(ctx).Model(&m).Updates(map[string]any{
		"name":       row.Name,
		"value":      row.Value,
		"sort_order": row.SortOrder,
	}).Error
}
