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
		RechargeBalance:  m.RechargeBalance,
		FrozenBalance:    m.FrozenBalance,
		FrozenIspay:      m.FrozenIspay,
		IspayBalance:     m.IspayBalance,
		LockBalance:      m.LockBalance,
		LockIspay:        m.LockIspay,
		PaidAmount:       m.PaidAmount,
		CapEffective:     m.CapEffective,
		DisabledAt:       m.DisabledAt,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

func (r *userRepo) FindByID(ctx context.Context, id uint64) (*biz.User, error) {
	var m UserModel
	if err := r.data.Session(ctx).First(&m, id).Error; err != nil {
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
		RechargeBalance:  u.RechargeBalance,
		FrozenBalance:    u.FrozenBalance,
		FrozenIspay:      u.FrozenIspay,
		IspayBalance:     u.IspayBalance,
		LockBalance:      u.LockBalance,
		LockIspay:        u.LockIspay,
		PaidAmount:       u.PaidAmount,
		CapEffective:     u.CapEffective,
	}
	if err := r.data.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	return r.FindByID(ctx, m.ID)
}

func (r *userRepo) AddPaidAmount(ctx context.Context, userID uint64, delta decimal.Decimal) error {
	res := r.data.Session(ctx).Model(&UserModel{}).
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
	res := r.data.Session(ctx).Model(&UserModel{}).
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

func (r *userRepo) ListByInviter(ctx context.Context, inviterID uint64) ([]*biz.User, error) {
	if inviterID == 0 {
		return []*biz.User{}, nil
	}
	var rows []UserModel
	if err := r.data.Session(ctx).Where("inviter_id = ?", inviterID).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.User, len(rows))
	for i := range rows {
		out[i] = toBizUser(&rows[i])
	}
	return out, nil
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

func (r *userRepo) AddRechargeBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error {
	res := r.data.Session(ctx).Model(&UserModel{}).Where("id = ?", userID).
		Update("recharge_balance", gorm.Expr("recharge_balance + ?", delta))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return biz.ErrUserNotFound
	}
	return nil
}

func (r *userRepo) SubRechargeBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error {
	res := r.data.Session(ctx).Model(&UserModel{}).
		Where("id = ? AND recharge_balance >= ?", userID, delta).
		Update("recharge_balance", gorm.Expr("recharge_balance - ?", delta))
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

func (r *userRepo) AddIspayBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error {
	res := r.data.Session(ctx).Model(&UserModel{}).Where("id = ?", userID).
		Update("ispay_balance", gorm.Expr("ispay_balance + ?", delta))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return biz.ErrUserNotFound
	}
	return nil
}

func (r *userRepo) SubIspayBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error {
	res := r.data.Session(ctx).Model(&UserModel{}).
		Where("id = ? AND ispay_balance >= ?", userID, delta).
		Update("ispay_balance", gorm.Expr("ispay_balance - ?", delta))
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

func (r *userRepo) AddLockBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error {
	res := r.data.Session(ctx).Model(&UserModel{}).Where("id = ?", userID).
		Update("lock_balance", gorm.Expr("lock_balance + ?", delta))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return biz.ErrUserNotFound
	}
	return nil
}

func (r *userRepo) SubLockBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error {
	res := r.data.Session(ctx).Model(&UserModel{}).
		Where("id = ? AND lock_balance >= ?", userID, delta).
		Update("lock_balance", gorm.Expr("lock_balance - ?", delta))
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

func (r *userRepo) AddLockIspay(ctx context.Context, userID uint64, delta decimal.Decimal) error {
	res := r.data.Session(ctx).Model(&UserModel{}).Where("id = ?", userID).
		Update("lock_ispay", gorm.Expr("lock_ispay + ?", delta))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return biz.ErrUserNotFound
	}
	return nil
}

func (r *userRepo) SubLockIspay(ctx context.Context, userID uint64, delta decimal.Decimal) error {
	res := r.data.Session(ctx).Model(&UserModel{}).
		Where("id = ? AND lock_ispay >= ?", userID, delta).
		Update("lock_ispay", gorm.Expr("lock_ispay - ?", delta))
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

func (r *userRepo) AddFrozenIspay(ctx context.Context, userID uint64, delta decimal.Decimal) error {
	res := r.data.Session(ctx).Model(&UserModel{}).Where("id = ?", userID).
		Update("frozen_ispay", gorm.Expr("frozen_ispay + ?", delta))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return biz.ErrUserNotFound
	}
	return nil
}

func (r *userRepo) SubFrozenIspay(ctx context.Context, userID uint64, delta decimal.Decimal) error {
	res := r.data.Session(ctx).Model(&UserModel{}).
		Where("id = ? AND frozen_ispay >= ?", userID, delta).
		Update("frozen_ispay", gorm.Expr("frozen_ispay - ?", delta))
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

func (r *ledgerRepo) ListPaged(ctx context.Context, address string, entryTypes []string, page, pageSize int) ([]*biz.LedgerEntry, int, error) {
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
	if len(entryTypes) == 1 {
		base = base.Where("ledger_entries.entry_type = ?", entryTypes[0])
	} else if len(entryTypes) > 1 {
		base = base.Where("ledger_entries.entry_type IN ?", entryTypes)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type ledgerAdminRow struct {
		LedgerEntryModel
		Address string `gorm:"column:address"`
	}
	var rows []ledgerAdminRow
	offset := (page - 1) * pageSize
	if err := base.Session(&gorm.Session{}).
		Select("ledger_entries.*, users.address AS address").
		Order("ledger_entries.created_at DESC, ledger_entries.id DESC").
		Offset(offset).Limit(pageSize).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.LedgerEntry, len(rows))
	for i := range rows {
		out[i] = ledgerModelToBiz(&rows[i].LedgerEntryModel)
		out[i].Address = rows[i].Address
	}
	return out, int(total), nil
}

func (r *ledgerRepo) ListByTypes(ctx context.Context, address string, entryTypes []string) ([]*biz.LedgerEntry, error) {
	base := r.data.db.WithContext(ctx).Table("ledger_entries").
		Joins("LEFT JOIN users ON users.id = ledger_entries.user_id")
	if address != "" {
		base = base.Where("users.address LIKE ?", "%"+address+"%")
	}
	if len(entryTypes) == 1 {
		base = base.Where("ledger_entries.entry_type = ?", entryTypes[0])
	} else if len(entryTypes) > 1 {
		base = base.Where("ledger_entries.entry_type IN ?", entryTypes)
	}
	type ledgerAdminRow struct {
		LedgerEntryModel
		Address string `gorm:"column:address"`
	}
	var rows []ledgerAdminRow
	if err := base.Session(&gorm.Session{}).
		Select("ledger_entries.*, users.address AS address").
		Order("ledger_entries.created_at DESC, ledger_entries.id DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.LedgerEntry, len(rows))
	for i := range rows {
		out[i] = ledgerModelToBiz(&rows[i].LedgerEntryModel)
		out[i].Address = rows[i].Address
	}
	return out, nil
}

func (r *ledgerRepo) FindMatching(ctx context.Context, userID uint64, entryType string, orderID *uint64, settleDate *time.Time, remark string) (*biz.LedgerEntry, error) {
	q := r.data.Session(ctx).Model(&LedgerEntryModel{}).
		Where("user_id = ? AND entry_type = ? AND remark = ?", userID, entryType, remark)
	if orderID != nil {
		q = q.Where("order_id = ?", *orderID)
	} else {
		q = q.Where("order_id IS NULL")
	}
	if settleDate != nil {
		q = q.Where("settle_date = ?", settleDate.Format("2006-01-02"))
	}
	var m LedgerEntryModel
	if err := q.Order("id ASC").First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return ledgerModelToBiz(&m), nil
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

func (r *ledgerRepo) ExistsByOrderTypeAndDate(ctx context.Context, orderID uint64, entryType string, settleDate time.Time) (bool, error) {
	var n int64
	err := r.data.Session(ctx).Model(&LedgerEntryModel{}).
		Where("order_id = ? AND entry_type = ? AND settle_date = ?", orderID, entryType, settleDate).
		Limit(1).Count(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *ledgerRepo) CountByOrderAndType(ctx context.Context, orderID uint64, entryType string) (int, error) {
	var n int64
	err := r.data.Session(ctx).Model(&LedgerEntryModel{}).
		Where("order_id = ? AND entry_type = ?", orderID, entryType).
		Count(&n).Error
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

func (r *ledgerRepo) ListByOrderIDsAndTypes(ctx context.Context, orderIDs []uint64, entryTypes []string) ([]*biz.LedgerEntry, error) {
	if len(orderIDs) == 0 || len(entryTypes) == 0 {
		return nil, nil
	}
	var rows []LedgerEntryModel
	err := r.data.Session(ctx).
		Where("order_id IN ? AND entry_type IN ?", orderIDs, entryTypes).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return ledgerModelsToBiz(rows), nil
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

func (r *ledgerRepo) ExistsByUserTypeDateRemark(ctx context.Context, userID uint64, entryType string, settleDate time.Time, remark string) (bool, error) {
	var n int64
	err := r.data.Session(ctx).Model(&LedgerEntryModel{}).
		Where("user_id = ? AND entry_type = ? AND settle_date = ? AND remark = ?", userID, entryType, settleDate, remark).
		Limit(1).Count(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *ledgerRepo) ListByTypeAndDate(ctx context.Context, entryType string, settleDate time.Time) ([]*biz.LedgerEntry, error) {
	var rows []LedgerEntryModel
	err := r.data.Session(ctx).
		Where("entry_type = ? AND settle_date = ?", entryType, settleDate).
		Order("id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return ledgerModelsToBiz(rows), nil
}

func ledgerModelToBiz(m *LedgerEntryModel) *biz.LedgerEntry {
	return &biz.LedgerEntry{
		ID:          m.ID,
		UserID:      m.UserID,
		OrderID:     m.OrderID,
		EntryType:   m.EntryType,
		Amount:      m.Amount,
		BalanceKind: m.BalanceKind,
		SettleDate:  m.SettleDate,
		Remark:      m.Remark,
		CreatedAt:   m.CreatedAt,
	}
}

func ledgerModelsToBiz(rows []LedgerEntryModel) []*biz.LedgerEntry {
	out := make([]*biz.LedgerEntry, len(rows))
	for i := range rows {
		out[i] = ledgerModelToBiz(&rows[i])
	}
	return out
}
