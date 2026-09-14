package biz

import (
	"context"
	"fmt"

	"github.com/cigc/app/internal/pkg/money"
	"github.com/cigc/app/internal/pkg/wallet"
	"github.com/shopspring/decimal"
)

const (
	AdjustAvailable = "available"
	AdjustRecharge  = "recharge"
	AdjustLock      = "lock"
	AdjustIspay     = "ispay"
	AdjustLockIspay = "lock_ispay"
)

// AdjustUseCase 管理端加减可提现 / 冻结收益（不含提现 frozen）。
type AdjustUseCase struct {
	users    UserRepo
	balances UserBalanceRepo
	ledger   LedgerRepo
	tx       TxRunner
}

// NewAdjustUseCase 构造调账用例。
func NewAdjustUseCase(users UserRepo, balances UserBalanceRepo, ledger LedgerRepo, tx TxRunner) *AdjustUseCase {
	if tx == nil {
		tx = NopTx{}
	}
	return &AdjustUseCase{users: users, balances: balances, ledger: ledger, tx: tx}
}

// NormalizeAdjustKind 只允许可调内部账户。
func NormalizeAdjustKind(kind string) (string, error) {
	switch kind {
	case AdjustAvailable, AdjustRecharge, AdjustLock, AdjustIspay, AdjustLockIspay:
		return kind, nil
	default:
		return "", ErrAdjustKind
	}
}

// Adjust 按地址加减一档余额，正加负减，余额不低于 0，并写 admin_adjust 流水。
func (uc *AdjustUseCase) Adjust(ctx context.Context, address, kind string, delta decimal.Decimal) (*User, error) {
	addr, ok := wallet.NormalizeAddress(address)
	if !ok {
		return nil, ErrUserNotFound
	}
	kind, err := NormalizeAdjustKind(kind)
	if err != nil {
		return nil, err
	}
	delta = money.Round(delta)
	if delta.IsZero() {
		return nil, ErrInvalidAmount
	}
	u, err := uc.users.FindByAddress(ctx, addr)
	if err != nil {
		return nil, err
	}
	if err := uc.tx.InTx(ctx, func(ctx context.Context) error {
		if err := uc.apply(ctx, u.ID, kind, delta); err != nil {
			return err
		}
		return uc.ledger.Create(ctx, &LedgerEntry{
			UserID:      u.ID,
			EntryType:   LedgerAdminAdjust,
			Amount:      delta,
			BalanceKind: kind,
			Remark:      fmt.Sprintf("admin adjust kind=%s", kind),
		})
	}); err != nil {
		return nil, err
	}
	return uc.users.FindByID(ctx, u.ID)
}

func (uc *AdjustUseCase) apply(ctx context.Context, userID uint64, kind string, delta decimal.Decimal) error {
	abs := delta
	if abs.IsNegative() {
		abs = abs.Neg()
	}
	switch kind {
	case AdjustAvailable:
		if delta.IsPositive() {
			return uc.balances.AddAvailableBalance(ctx, userID, abs)
		}
		return uc.balances.SubAvailableBalance(ctx, userID, abs)
	case AdjustRecharge:
		if delta.IsPositive() {
			return uc.balances.AddRechargeBalance(ctx, userID, abs)
		}
		return uc.balances.SubRechargeBalance(ctx, userID, abs)
	case AdjustLock:
		if delta.IsPositive() {
			return uc.balances.AddLockBalance(ctx, userID, abs)
		}
		return uc.balances.SubLockBalance(ctx, userID, abs)
	case AdjustIspay:
		if delta.IsPositive() {
			return uc.balances.AddIspayBalance(ctx, userID, abs)
		}
		return uc.balances.SubIspayBalance(ctx, userID, abs)
	case AdjustLockIspay:
		if delta.IsPositive() {
			return uc.balances.AddLockIspay(ctx, userID, abs)
		}
		return uc.balances.SubLockIspay(ctx, userID, abs)
	default:
		return ErrAdjustKind
	}
}
