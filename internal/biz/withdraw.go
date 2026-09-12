package biz

import (
	"context"
	"time"

	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
)

const (
	WithdrawPending   = "pending"
	WithdrawRewarded  = "rewarded"
	WithdrawDoing     = "doing"
	WithdrawPass      = "pass"
	WithdrawRejected  = "rejected"
	WithdrawCancelled = "cancelled"

	ConfigMinWithdraw  = "min_withdraw_amount"
	ConfigDirectRate   = "direct_rate"
	ConfigMatchRate    = "match_rate"
	defaultMinWithdraw = "10"
	defaultDirectRate  = "0.10"
	defaultMatchRate   = "0.10"
)

// Withdraw 是内部账户提现单。P0 不打款：pending 冻结，rewarded 已审待打款。
type Withdraw struct {
	ID             uint64
	UserID         uint64
	Amount         decimal.Decimal
	FeeAmount      decimal.Decimal
	CreditedAmount decimal.Decimal
	Status         string
	Remark         string
	TxHash         string
	PayoutError    string
	ReviewedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AdminWithdrawRow 管理端提现行。
type AdminWithdrawRow struct {
	Withdraw
	Address string
}

// WithdrawPage 提现分页。
type WithdrawPage struct {
	Items []*Withdraw
	Total int
}

// AdminWithdrawPage 管理端提现分页。
type AdminWithdrawPage struct {
	Items []*AdminWithdrawRow
	Total int
}

// WithdrawRepo 提现单读写。
type WithdrawRepo interface {
	Create(ctx context.Context, w *Withdraw) (*Withdraw, error)
	FindByID(ctx context.Context, id uint64) (*Withdraw, error)
	CasStatus(ctx context.Context, id uint64, from, to, remark string, reviewedAt *time.Time) error
	ListByUser(ctx context.Context, userID uint64) ([]*Withdraw, error)
	ListAdmin(ctx context.Context, address, status string, page, pageSize int) ([]*AdminWithdrawRow, int, error)
}

// ConfigRepo 读取 business_configs。
type ConfigRepo interface {
	GetValue(ctx context.Context, key string) (string, error)
}

// TxRunner 把一组写操作放进同一事务；测试可用空实现。
type TxRunner interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// NopTx 不包事务，直接执行（单元测试）。
type NopTx struct{}

// InTx 直接调用 fn。
func (NopTx) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// WithdrawUseCase 提现申请与审核（不做链上打款）。
type WithdrawUseCase struct {
	users     UserRepo
	balances  UserBalanceRepo
	ledger    LedgerRepo
	withdraws WithdrawRepo
	configs   ConfigRepo
	tx        TxRunner
	now       func() time.Time
}

// NewWithdrawUseCase 构造提现用例。
func NewWithdrawUseCase(
	users UserRepo,
	balances UserBalanceRepo,
	ledger LedgerRepo,
	withdraws WithdrawRepo,
	configs ConfigRepo,
	tx TxRunner,
) *WithdrawUseCase {
	if tx == nil {
		tx = NopTx{}
	}
	return &WithdrawUseCase{
		users:     users,
		balances:  balances,
		ledger:    ledger,
		withdraws: withdraws,
		configs:   configs,
		tx:        tx,
		now:       time.Now,
	}
}

func (uc *WithdrawUseCase) minAmount(ctx context.Context) decimal.Decimal {
	raw := defaultMinWithdraw
	if uc.configs != nil {
		if v, err := uc.configs.GetValue(ctx, ConfigMinWithdraw); err == nil && v != "" {
			raw = v
		}
	}
	d, err := decimal.NewFromString(raw)
	if err != nil {
		d = decimal.RequireFromString(defaultMinWithdraw)
	}
	return money.Round(d)
}

// Create 从 available 扣到 frozen，写冻结流水，状态 pending。手续费本刀为 0。
func (uc *WithdrawUseCase) Create(ctx context.Context, userID uint64, amount decimal.Decimal) (*Withdraw, error) {
	amount = money.Round(amount)
	if !amount.IsPositive() {
		return nil, ErrInvalidAmount
	}
	min := uc.minAmount(ctx)
	if amount.LessThan(min) {
		return nil, ErrWithdrawBelowMin
	}
	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.IsDisabled() {
		return nil, ErrUserDisabled
	}
	if money.Round(user.AvailableBalance).LessThan(amount) {
		return nil, ErrInsufficientBalance
	}

	var created *Withdraw
	err = uc.tx.InTx(ctx, func(ctx context.Context) error {
		if err := uc.balances.SubAvailableBalance(ctx, userID, amount); err != nil {
			return err
		}
		if err := uc.balances.AddFrozenBalance(ctx, userID, amount); err != nil {
			return err
		}
		neg := amount.Neg()
		if err := uc.ledger.Create(ctx, &LedgerEntry{
			UserID: userID, EntryType: LedgerFreeze, Amount: neg,
			BalanceKind: BalanceAvailable, Remark: "withdraw freeze",
		}); err != nil {
			return err
		}
		if err := uc.ledger.Create(ctx, &LedgerEntry{
			UserID: userID, EntryType: LedgerFreeze, Amount: amount,
			BalanceKind: BalanceFrozen, Remark: "withdraw freeze",
		}); err != nil {
			return err
		}
		w, err := uc.withdraws.Create(ctx, &Withdraw{
			UserID:         userID,
			Amount:         amount,
			FeeAmount:      decimal.Zero,
			CreditedAmount: amount,
			Status:         WithdrawPending,
		})
		if err != nil {
			return err
		}
		created = w
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// ListUser 用户提现列表后分页。
func (uc *WithdrawUseCase) ListUser(ctx context.Context, userID uint64, page int) (*WithdrawPage, error) {
	rows, err := uc.withdraws.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []*Withdraw{}
	}
	items := paginateWithdraws(rows, page, DefaultRewardPageSize)
	return &WithdrawPage{Items: items, Total: len(rows)}, nil
}

func paginateWithdraws(items []*Withdraw, page, pageSize int) []*Withdraw {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultRewardPageSize
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []*Withdraw{}
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

// ListAdmin 管理端分页，可筛 address、status。
func (uc *WithdrawUseCase) ListAdmin(ctx context.Context, address, status string, page int) (*AdminWithdrawPage, error) {
	if page < 1 {
		page = 1
	}
	rows, total, err := uc.withdraws.ListAdmin(ctx, address, status, page, DefaultRewardPageSize)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []*AdminWithdrawRow{}
	}
	return &AdminWithdrawPage{Items: rows, Total: total}, nil
}

// Pass pending → rewarded，冻结保持，等待后续打款模块。
func (uc *WithdrawUseCase) Pass(ctx context.Context, id uint64) (*Withdraw, error) {
	if id == 0 {
		return nil, ErrInvalidAmount
	}
	var out *Withdraw
	err := uc.tx.InTx(ctx, func(ctx context.Context) error {
		w, err := uc.withdraws.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if w.Status != WithdrawPending {
			return ErrWithdrawConflict
		}
		now := uc.now()
		if err := uc.withdraws.CasStatus(ctx, id, WithdrawPending, WithdrawRewarded, w.Remark, &now); err != nil {
			return err
		}
		w.Status = WithdrawRewarded
		w.ReviewedAt = &now
		out = w
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Reject pending → rejected，frozen 退回 available，写解冻流水。
func (uc *WithdrawUseCase) Reject(ctx context.Context, id uint64) (*Withdraw, error) {
	if id == 0 {
		return nil, ErrInvalidAmount
	}
	var out *Withdraw
	err := uc.tx.InTx(ctx, func(ctx context.Context) error {
		w, err := uc.withdraws.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if w.Status != WithdrawPending {
			return ErrWithdrawConflict
		}
		amount := money.Round(w.Amount)
		if err := uc.balances.SubFrozenBalance(ctx, w.UserID, amount); err != nil {
			return err
		}
		if err := uc.balances.AddAvailableBalance(ctx, w.UserID, amount); err != nil {
			return err
		}
		neg := amount.Neg()
		if err := uc.ledger.Create(ctx, &LedgerEntry{
			UserID: w.UserID, EntryType: LedgerUnfreeze, Amount: neg,
			BalanceKind: BalanceFrozen, Remark: "withdraw reject unfreeze",
		}); err != nil {
			return err
		}
		if err := uc.ledger.Create(ctx, &LedgerEntry{
			UserID: w.UserID, EntryType: LedgerUnfreeze, Amount: amount,
			BalanceKind: BalanceAvailable, Remark: "withdraw reject unfreeze",
		}); err != nil {
			return err
		}
		now := uc.now()
		if err := uc.withdraws.CasStatus(ctx, id, WithdrawPending, WithdrawRejected, w.Remark, &now); err != nil {
			return err
		}
		w.Status = WithdrawRejected
		w.ReviewedAt = &now
		out = w
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
