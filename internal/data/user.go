package data

import (
	"context"
	"errors"
	"time"

	"github.com/cigc/app/internal/biz"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type userRepo struct{ data *Data }

func NewUserRepo(d *Data) biz.UserRepo { return &userRepo{data: d} }

func toBizUser(m *UserModel) *biz.User {
	return &biz.User{
		ID:               m.ID,
		Address:          m.Address,
		InviterID:        m.InviterID,
		AvailableBalance: m.AvailableBalance,
		FrozenBalance:    m.FrozenBalance,
		PaidAmount:       m.PaidAmount,
		CapEffective:     m.CapEffective,
		DisabledAt:       m.DisabledAt,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

func (r *userRepo) FindByID(ctx context.Context, id uint64) (*biz.User, error) {
	var m UserModel
	if err := r.data.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrUserNotFound
		}
		return nil, err
	}
	return toBizUser(&m), nil
}

func (r *userRepo) FindByAddress(ctx context.Context, address string) (*biz.User, error) {
	var m UserModel
	if err := r.data.db.WithContext(ctx).Where("address = ?", address).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrUserNotFound
		}
		return nil, err
	}
	return toBizUser(&m), nil
}

func (r *userRepo) Create(ctx context.Context, u *biz.User) (*biz.User, error) {
	m := UserModel{
		Address:          u.Address,
		InviterID:        u.InviterID,
		AvailableBalance: u.AvailableBalance,
		FrozenBalance:    u.FrozenBalance,
		PaidAmount:       u.PaidAmount,
		CapEffective:     u.CapEffective,
	}
	if err := r.data.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	return r.FindByID(ctx, m.ID)
}

func (r *userRepo) AddPaidAmount(ctx context.Context, userID uint64, delta decimal.Decimal) error {
	res := r.data.db.WithContext(ctx).Model(&UserModel{}).
		Where("id = ?", userID).
		Update("paid_amount", gorm.Expr("paid_amount + ?", delta))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return biz.ErrUserNotFound
	}
	return nil
}

func (r *userRepo) ListAll(ctx context.Context) ([]*biz.User, error) {
	var rows []UserModel
	if err := r.data.db.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.User, len(rows))
	for i := range rows {
		out[i] = toBizUser(&rows[i])
	}
	return out, nil
}

func (r *userRepo) SetCapEffective(ctx context.Context, userID uint64, cap decimal.Decimal) error {
	res := r.data.db.WithContext(ctx).Model(&UserModel{}).
		Where("id = ?", userID).
		Update("cap_effective", cap)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return biz.ErrUserNotFound
	}
	return nil
}

func (r *userRepo) ListAdmin(ctx context.Context, address string, page, pageSize int) ([]*biz.User, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = biz.DefaultRewardPageSize
	}
	q := r.data.db.WithContext(ctx).Model(&UserModel{})
	if address != "" {
		q = q.Where("address LIKE ?", "%"+address+"%")
	}
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []UserModel
	offset := (page - 1) * pageSize
	if err := q.Session(&gorm.Session{}).
		Order("id DESC").
		Offset(offset).Limit(pageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.User, len(rows))
	for i := range rows {
		out[i] = toBizUser(&rows[i])
	}
	return out, int(total), nil
}

func (r *userRepo) SetDisabledAt(ctx context.Context, userID uint64, disabledAt *time.Time) error {
	res := r.data.db.WithContext(ctx).Model(&UserModel{}).
		Where("id = ?", userID).
		Update("disabled_at", disabledAt)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return biz.ErrUserNotFound
	}
	return nil
}

func NewUserBalanceRepo(d *Data) biz.UserBalanceRepo { return &userRepo{data: d} }

func (r *userRepo) AddAvailableBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error {
	res := r.data.Session(ctx).Model(&UserModel{}).Where("id = ?", userID).
		Update("available_balance", gorm.Expr("available_balance + ?", delta))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return biz.ErrUserNotFound
	}
	return nil
}

func (r *userRepo) SubAvailableBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error {
	res := r.data.Session(ctx).Model(&UserModel{}).
		Where("id = ? AND available_balance >= ?", userID, delta).
		Update("available_balance", gorm.Expr("available_balance - ?", delta))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		if _, err := r.FindByID(ctx, userID); err != nil {
			return err
		}
		return biz.ErrInsufficientBalance
	}
	return nil
}

func (r *userRepo) AddFrozenBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error {
	res := r.data.Session(ctx).Model(&UserModel{}).Where("id = ?", userID).
		Update("frozen_balance", gorm.Expr("frozen_balance + ?", delta))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return biz.ErrUserNotFound
	}
	return nil
}

func (r *userRepo) SubFrozenBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error {
	res := r.data.Session(ctx).Model(&UserModel{}).
		Where("id = ? AND frozen_balance >= ?", userID, delta).
		Update("frozen_balance", gorm.Expr("frozen_balance - ?", delta))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		if _, err := r.FindByID(ctx, userID); err != nil {
			return err
		}
		return biz.ErrInsufficientBalance
	}
	return nil
}

type recommendRepo struct{ data *Data }

func NewRecommendRepo(d *Data) biz.RecommendRepo { return &recommendRepo{data: d} }

func (r *recommendRepo) GetPath(ctx context.Context, userID uint64) (string, error) {
	var m UserRecommendModel
	err := r.data.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return m.Path, nil
}

func (r *recommendRepo) SavePath(ctx context.Context, userID uint64, path string) error {
	var m UserRecommendModel
	err := r.data.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.data.db.WithContext(ctx).Create(&UserRecommendModel{UserID: userID, Path: path}).Error
	}
	if err != nil {
		return err
	}
	return r.data.db.WithContext(ctx).Model(&m).Update("path", path).Error
}

type ledgerRepo struct{ data *Data }

func NewLedgerRepo(d *Data) biz.LedgerRepo { return &ledgerRepo{data: d} }

func (r *ledgerRepo) Create(ctx context.Context, e *biz.LedgerEntry) error {
	m := LedgerEntryModel{
		UserID:      e.UserID,
		OrderID:     e.OrderID,
		EntryType:   e.EntryType,
		Amount:      e.Amount,
		BalanceKind: e.BalanceKind,
		SettleDate:  e.SettleDate,
		Remark:      e.Remark,
	}
	return r.data.Session(ctx).Create(&m).Error
}

func (r *ledgerRepo) ListByUser(ctx context.Context, userID uint64, from, to time.Time) ([]*biz.LedgerEntry, error) {
	q := r.data.db.WithContext(ctx).Where("created_at >= ? AND created_at < ?", from, to)
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	var rows []LedgerEntryModel
	if err := q.Order("created_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return ledgerModelsToBiz(rows), nil
}

func (r *ledgerRepo) ListPaged(ctx context.Context, address, entryType string, page, pageSize int) ([]*biz.LedgerEntry, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	base := r.data.db.WithContext(ctx).Table("ledger_entries").
		Joins("LEFT JOIN users ON users.id = ledger_entries.user_id")
	if address != "" {
		base = base.Where("users.address LIKE ?", "%"+address+"%")
	}
	if entryType != "" {
		base = base.Where("ledger_entries.entry_type = ?", entryType)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []LedgerEntryModel
	offset := (page - 1) * pageSize
	if err := base.Session(&gorm.Session{}).
		Select("ledger_entries.*").
		Order("ledger_entries.created_at DESC, ledger_entries.id DESC").
		Offset(offset).Limit(pageSize).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return ledgerModelsToBiz(rows), int(total), nil
}

func (r *ledgerRepo) ExistsByOrderAndType(ctx context.Context, orderID uint64, entryType string) (bool, error) {
	var n int64
	err := r.data.Session(ctx).Model(&LedgerEntryModel{}).
		Where("order_id = ? AND entry_type = ?", orderID, entryType).
		Limit(1).Count(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *ledgerRepo) ExistsByUserTypeAndDate(ctx context.Context, userID uint64, entryType string, settleDate time.Time) (bool, error) {
	var n int64
	err := r.data.Session(ctx).Model(&LedgerEntryModel{}).
		Where("user_id = ? AND entry_type = ? AND settle_date = ?", userID, entryType, settleDate).
		Limit(1).Count(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func ledgerModelsToBiz(rows []LedgerEntryModel) []*biz.LedgerEntry {
	out := make([]*biz.LedgerEntry, len(rows))
	for i := range rows {
		out[i] = &biz.LedgerEntry{
			ID:          rows[i].ID,
			UserID:      rows[i].UserID,
			OrderID:     rows[i].OrderID,
			EntryType:   rows[i].EntryType,
			Amount:      rows[i].Amount,
			BalanceKind: rows[i].BalanceKind,
			SettleDate:  rows[i].SettleDate,
			Remark:      rows[i].Remark,
			CreatedAt:   rows[i].CreatedAt,
		}
	}
	return out
}
