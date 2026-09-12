package data

import (
	"context"
	"errors"
	"time"

	"github.com/cigc/app/internal/biz"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type packageRepo struct{ data *Data }

func NewPackageRepo(d *Data) biz.PackageRepo { return &packageRepo{data: d} }

func toBizPackage(m *PackageModel) *biz.Package {
	return &biz.Package{
		ID:        m.ID,
		Amount:    m.Amount,
		Title:     m.Title,
		GoodsDesc: m.GoodsDesc,
		DailyCap:  m.DailyCap,
		SortOrder: m.SortOrder,
		Enabled:   m.Enabled,
	}
}

func (r *packageRepo) ListEnabled(ctx context.Context) ([]*biz.Package, error) {
	var rows []PackageModel
	if err := r.data.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.Package, len(rows))
	for i := range rows {
		out[i] = toBizPackage(&rows[i])
	}
	return out, nil
}

func (r *packageRepo) ListAll(ctx context.Context) ([]*biz.Package, error) {
	var rows []PackageModel
	if err := r.data.db.WithContext(ctx).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.Package, len(rows))
	for i := range rows {
		out[i] = toBizPackage(&rows[i])
	}
	return out, nil
}

func (r *packageRepo) FindByAmount(ctx context.Context, amount decimal.Decimal) (*biz.Package, error) {
	var m PackageModel
	if err := r.data.db.WithContext(ctx).Where("amount = ?", amount).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrPackageNotFound
		}
		return nil, err
	}
	return toBizPackage(&m), nil
}

func (r *packageRepo) FindByID(ctx context.Context, id uint64) (*biz.Package, error) {
	var m PackageModel
	if err := r.data.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrPackageNotFound
		}
		return nil, err
	}
	return toBizPackage(&m), nil
}

type orderRepo struct{ data *Data }

func NewOrderRepo(d *Data) biz.OrderRepo { return &orderRepo{data: d} }

func toBizOrder(m *OrderModel) *biz.Order {
	return &biz.Order{
		ID:            m.ID,
		UserID:        m.UserID,
		PackageID:     m.PackageID,
		Amount:        m.Amount,
		TitleSnapshot: m.TitleSnapshot,
		GoodsSnapshot: m.GoodsSnapshot,
		Status:        m.Status,
		TxHash:        m.TxHash,
		LogIndex:      m.LogIndex,
		PaidAt:        m.PaidAt,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

func (r *orderRepo) Create(ctx context.Context, o *biz.Order) (*biz.Order, error) {
	m := OrderModel{
		UserID:        o.UserID,
		PackageID:     o.PackageID,
		Amount:        o.Amount,
		TitleSnapshot: o.TitleSnapshot,
		GoodsSnapshot: o.GoodsSnapshot,
		Status:        o.Status,
		TxHash:        o.TxHash,
		LogIndex:      o.LogIndex,
	}
	if err := r.data.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	return r.FindByID(ctx, m.ID)
}

func (r *orderRepo) FindByID(ctx context.Context, id uint64) (*biz.Order, error) {
	var m OrderModel
	if err := r.data.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrOrderNotFound
		}
		return nil, err
	}
	return toBizOrder(&m), nil
}

func (r *orderRepo) ListByUser(ctx context.Context, userID uint64) ([]*biz.Order, error) {
	var rows []OrderModel
	if err := r.data.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.Order, len(rows))
	for i := range rows {
		out[i] = toBizOrder(&rows[i])
	}
	return out, nil
}

func (r *orderRepo) MarkPaid(ctx context.Context, id uint64, userID uint64, amount decimal.Decimal, paidAt time.Time) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&OrderModel{}).
			Where("id = ? AND status = ?", id, biz.OrderPending).
			Updates(map[string]any{
				"status":  biz.OrderPaid,
				"paid_at": paidAt,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return biz.ErrOrderConflict
		}
		res = tx.Model(&UserModel{}).
			Where("id = ?", userID).
			Update("paid_amount", gorm.Expr("paid_amount + ?", amount))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return biz.ErrUserNotFound
		}
		return nil
	})
}

func (r *orderRepo) ListAdmin(ctx context.Context, address, status string, page, pageSize int) ([]*biz.AdminOrderRow, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = biz.DefaultRewardPageSize
	}
	base := r.data.db.WithContext(ctx).Table("orders").
		Joins("LEFT JOIN users ON users.id = orders.user_id")
	if address != "" {
		base = base.Where("users.address LIKE ?", "%"+address+"%")
	}
	if status != "" {
		base = base.Where("orders.status = ?", status)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type row struct {
		OrderModel
		Address string `gorm:"column:address"`
	}
	var rows []row
	offset := (page - 1) * pageSize
	if err := base.Session(&gorm.Session{}).
		Select("orders.*, users.address AS address").
		Order("orders.id DESC").
		Offset(offset).Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminOrderRow, len(rows))
	for i := range rows {
		o := toBizOrder(&rows[i].OrderModel)
		out[i] = &biz.AdminOrderRow{Order: *o, Address: rows[i].Address}
	}
	return out, int(total), nil
}

func (r *orderRepo) ListPaidBetween(ctx context.Context, from, to time.Time) ([]*biz.Order, error) {
	var rows []OrderModel
	if err := r.data.Session(ctx).
		Where("status = ? AND paid_at >= ? AND paid_at < ?", biz.OrderPaid, from, to).
		Order("id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.Order, len(rows))
	for i := range rows {
		out[i] = toBizOrder(&rows[i])
	}
	return out, nil
}
