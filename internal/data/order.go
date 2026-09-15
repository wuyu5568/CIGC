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

type packageRepo struct{ data *Data }

func NewPackageRepo(d *Data) biz.PackageRepo { return &packageRepo{data: d} }

func toBizPackage(m *PackageModel) *biz.Package {
	return &biz.Package{
		ID:          m.ID,
		Amount:      m.Amount,
		Title:       m.Title,
		GoodsDesc:   m.GoodsDesc,
		DailyCap:    m.DailyCap,
		ReleaseDays: m.ReleaseDays,
		SortOrder:   m.SortOrder,
		Enabled:     m.Enabled,
		Image:       m.Image,
		Detail:      m.Detail,
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

func (r *packageRepo) FindByAmount(ctx context.Context, amount decimal.Decimal, days int) (*biz.Package, error) {
	if !biz.ValidReleaseDays(days) {
		days = biz.ReleaseDays300
	}
	var m PackageModel
	if err := r.data.db.WithContext(ctx).Where("amount = ? AND release_days = ?", amount, days).First(&m).Error; err != nil {
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

func (r *packageRepo) Update(ctx context.Context, p *biz.Package) (*biz.Package, error) {
	if p == nil || p.ID == 0 {
		return nil, biz.ErrPackageNotFound
	}
	res := r.data.Session(ctx).Model(&PackageModel{}).Where("id = ?", p.ID).Updates(map[string]any{
		"amount":       p.Amount,
		"title":        p.Title,
		"goods_desc":   p.GoodsDesc,
		"daily_cap":    p.DailyCap,
		"release_days": p.ReleaseDays,
		"sort_order":   p.SortOrder,
		"enabled":      p.Enabled,
		"image":        p.Image,
		"detail":       p.Detail,
	})
	if res.Error != nil {
		if isDuplicateKey(res.Error) {
			return nil, biz.ErrPackageAmountTaken
		}
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, biz.ErrPackageNotFound
	}
	return r.FindByID(ctx, p.ID)
}

func (r *packageRepo) Create(ctx context.Context, p *biz.Package) (*biz.Package, error) {
	m := PackageModel{
		Amount:      p.Amount,
		Title:       p.Title,
		GoodsDesc:   p.GoodsDesc,
		DailyCap:    p.DailyCap,
		ReleaseDays: p.ReleaseDays,
		SortOrder:   p.SortOrder,
		Enabled:     p.Enabled,
		Image:       p.Image,
		Detail:      p.Detail,
	}
	if err := r.data.Session(ctx).Create(&m).Error; err != nil {
		if isDuplicateKey(err) {
			return nil, biz.ErrPackageAmountTaken
		}
		return nil, err
	}
	return toBizPackage(&m), nil
}

func (r *packageRepo) Delete(ctx context.Context, id uint64) error {
	res := r.data.Session(ctx).Delete(&PackageModel{}, id)
	if res.Error != nil {
		if isForeignKey(res.Error) {
			return biz.ErrPackageInUse
		}
		return res.Error
	}
	if res.RowsAffected == 0 {
		return biz.ErrPackageNotFound
	}
	return nil
}

type orderRepo struct{ data *Data }

func NewOrderRepo(d *Data) biz.OrderRepo { return &orderRepo{data: d} }

func toBizOrder(m *OrderModel) *biz.Order {
	return &biz.Order{
		ID:            m.ID,
		OrderNo:       m.OrderNo,
		UserID:        m.UserID,
		PackageID:     m.PackageID,
		Amount:        m.Amount,
		TitleSnapshot: m.TitleSnapshot,
		GoodsSnapshot: m.GoodsSnapshot,
		Status:        m.Status,
		ReleaseDays:   m.ReleaseDays,
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
		ReleaseDays:   o.ReleaseDays,
		TxHash:        o.TxHash,
		LogIndex:      o.LogIndex,
	}
	if err := r.data.Session(ctx).Omit("order_no").Create(&m).Error; err != nil {
		return nil, err
	}
	fixed := strings.TrimSpace(o.OrderNo)
	var last error
	for i := 0; i < 24; i++ {
		no := fixed
		if no == "" {
			no = biz.RandomOrderNo()
		}
		if err := r.data.Session(ctx).Model(&m).Update("order_no", no).Error; err != nil {
			last = err
			if fixed != "" || !isDuplicateKey(err) {
				return nil, err
			}
			continue
		}
		m.OrderNo = no
		return toBizOrder(&m), nil
	}
	if last == nil {
		last = errors.New("allocate order_no")
	}
	return nil, last
}

func (r *orderRepo) FindByID(ctx context.Context, id uint64) (*biz.Order, error) {
	var m OrderModel
	if err := r.data.Session(ctx).First(&m, id).Error; err != nil {
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
	return r.markPaid(ctx, id, userID, amount, paidAt, nil, 0)
}

func (r *orderRepo) MarkPaidWithChain(ctx context.Context, id uint64, userID uint64, amount decimal.Decimal, paidAt time.Time, txHash string, logIndex int) error {
	h := strings.ToLower(strings.TrimSpace(txHash))
	return r.markPaid(ctx, id, userID, amount, paidAt, &h, logIndex)
}

func (r *orderRepo) markPaid(ctx context.Context, id uint64, userID uint64, amount decimal.Decimal, paidAt time.Time, txHash *string, logIndex int) error {
	return r.data.Session(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"status":  biz.OrderPaid,
			"paid_at": paidAt,
		}
		if txHash != nil && *txHash != "" {
			updates["tx_hash"] = *txHash
			updates["log_index"] = logIndex
		}
		res := tx.Model(&OrderModel{}).
			Where("id = ? AND status = ?", id, biz.OrderPending).
			Updates(updates)
		if res.Error != nil {
			if isDuplicateKey(res.Error) {
				return biz.ErrOrderConflict
			}
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

func (r *orderRepo) FindOldestPendingByUserAmount(ctx context.Context, userID uint64, amount decimal.Decimal) (*biz.Order, error) {
	var m OrderModel
	err := r.data.Session(ctx).
		Where("user_id = ? AND status = ? AND amount = ?", userID, biz.OrderPending, amount).
		Order("id ASC").
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toBizOrder(&m), nil
}

func (r *orderRepo) FindByTxEvent(ctx context.Context, txHash string, logIndex int) (*biz.Order, error) {
	txHash = strings.ToLower(strings.TrimSpace(txHash))
	if txHash == "" {
		return nil, nil
	}
	var m OrderModel
	err := r.data.Session(ctx).
		Where("tx_hash = ? AND log_index = ?", txHash, logIndex).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toBizOrder(&m), nil
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

func (r *orderRepo) MaxPaidAmountByUser(ctx context.Context) (map[uint64]decimal.Decimal, error) {
	type row struct {
		UserID uint64
		MaxAmt decimal.Decimal
	}
	var rows []row
	err := r.data.Session(ctx).Model(&OrderModel{}).
		Select("user_id AS user_id, MAX(amount) AS max_amt").
		Where("status = ?", biz.OrderPaid).
		Group("user_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uint64]decimal.Decimal, len(rows))
	for _, it := range rows {
		out[it.UserID] = it.MaxAmt
	}
	return out, nil
}

func (r *orderRepo) ListPaidBefore(ctx context.Context, to time.Time) ([]*biz.Order, error) {
	var rows []OrderModel
	if err := r.data.Session(ctx).
		Where("status = ? AND paid_at IS NOT NULL AND paid_at < ?", biz.OrderPaid, to).
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
