package data

import (
	"context"
	"errors"
	"time"

	"github.com/cigc/app/internal/biz"
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
		Status:         w.Status,
		Remark:         w.Remark,
		TxHash:         w.TxHash,
		PayoutError:    w.PayoutError,
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

func (r *withdrawRepo) ListAdmin(ctx context.Context, address, status string, page, pageSize int) ([]*biz.AdminWithdrawRow, int, error) {
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
