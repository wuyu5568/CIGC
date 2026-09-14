package biz

import (
	"context"
	"fmt"
	"time"

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
	daily    DailyCapRepo
	tx       TxRunner
	now      func() time.Time
}

// NewAdjustUseCase 构造调账用例。
func NewAdjustUseCase(users UserRepo, balances UserBalanceRepo, ledger LedgerRepo, tx TxRunner, daily DailyCapRepo) *AdjustUseCase {
	if tx == nil {
		tx = NopTx{}
	}
	return &AdjustUseCase{users: users, balances: balances, ledger: ledger, daily: daily, tx: tx, now: time.Now}
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

func (uc *AdjustUseCase) nowTime() time.Time {
	if uc != nil && uc.now != nil {
		return uc.now()
	}
	return time.Now()
}

func adjustSettleDay(t time.Time) time.Time {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	t = t.In(loc)
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

// Adjust 按地址加减一档余额，正加负减，余额不低于 0，并写 admin_adjust 流水。
// 加 lock / lock_ispay 记冻结批次（次日 0:00 封账，再过 overflow_clear_hours 小时清除）；减则按 FIFO 扣批次。
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
			if err := uc.balances.AddLockBalance(ctx, userID, abs); err != nil {
				return err
			}
			return uc.addLockHold(ctx, userID, abs, decimal.Zero)
		}
		if err := uc.balances.SubLockBalance(ctx, userID, abs); err != nil {
			return err
		}
		return uc.debitLockHolds(ctx, userID, abs, decimal.Zero)
	case AdjustIspay:
		if delta.IsPositive() {
			return uc.balances.AddIspayBalance(ctx, userID, abs)
		}
		return uc.balances.SubIspayBalance(ctx, userID, abs)
	case AdjustLockIspay:
		if delta.IsPositive() {
			if err := uc.balances.AddLockIspay(ctx, userID, abs); err != nil {
				return err
			}
			return uc.addLockHold(ctx, userID, decimal.Zero, abs)
		}
		if err := uc.balances.SubLockIspay(ctx, userID, abs); err != nil {
			return err
		}
		return uc.debitLockHolds(ctx, userID, decimal.Zero, abs)
	default:
		return ErrAdjustKind
	}
}

func (uc *AdjustUseCase) addLockHold(ctx context.Context, userID uint64, usdt, ispay decimal.Decimal) error {
	if uc == nil || uc.daily == nil {
		return nil
	}
	usdt = money.Round(usdt)
	ispay = money.Round(ispay)
	if !usdt.IsPositive() && !ispay.IsPositive() {
		return nil
	}
	value := usdt
	if ispay.IsPositive() && !usdt.IsPositive() {
		value = money.Round(ispay.Mul(IspaySpotFallback()))
	}
	if !value.IsPositive() {
		value = ispay
	}
	now := uc.nowTime()
	return uc.daily.CreateHold(ctx, &CapOverflowHold{
		UserID:     userID,
		Value:      value,
		USDT:       usdt,
		Ispay:      ispay,
		SourceType: overflowSourceAdmin,
		SettleDate: adjustSettleDay(now),
		Remark:     remarkAdminLockHold,
		CreatedAt:  now,
	})
}

func (uc *AdjustUseCase) debitLockHolds(ctx context.Context, userID uint64, usdt, ispay decimal.Decimal) error {
	if uc == nil || uc.daily == nil {
		return nil
	}
	usdt = money.Round(usdt)
	ispay = money.Round(ispay)
	if !usdt.IsPositive() && !ispay.IsPositive() {
		return nil
	}
	holds, err := uc.daily.ListActiveHolds(ctx, userID)
	if err != nil {
		return err
	}
	remainU, remainI := usdt, ispay
	now := uc.nowTime()
	for _, h := range holds {
		if h == nil || (!remainU.IsPositive() && !remainI.IsPositive()) {
			break
		}
		cur, err := uc.daily.GetHold(ctx, h.ID)
		if err != nil {
			return err
		}
		if cur == nil || cur.ReleasedAt != nil || cur.BurnedAt != nil {
			continue
		}
		takeU := decimal.Zero
		if remainU.IsPositive() && cur.USDT.IsPositive() {
			takeU = cur.USDT
			if takeU.GreaterThan(remainU) {
				takeU = remainU
			}
		}
		takeI := decimal.Zero
		if remainI.IsPositive() && cur.Ispay.IsPositive() {
			takeI = cur.Ispay
			if takeI.GreaterThan(remainI) {
				takeI = remainI
			}
		}
		if !takeU.IsPositive() && !takeI.IsPositive() {
			continue
		}
		takeVal := decimal.Zero
		if cur.Value.IsPositive() && cur.USDT.IsPositive() && takeU.IsPositive() {
			takeVal = money.Round(cur.Value.Mul(takeU).Div(cur.USDT))
		}
		if cur.Value.IsPositive() && cur.Ispay.IsPositive() && takeI.IsPositive() && !cur.USDT.IsPositive() {
			takeVal = money.Round(cur.Value.Mul(takeI).Div(cur.Ispay))
		}
		cur.USDT = money.Round(cur.USDT.Sub(takeU))
		cur.Ispay = money.Round(cur.Ispay.Sub(takeI))
		cur.Value = money.Round(cur.Value.Sub(takeVal))
		if cur.USDT.IsNegative() {
			cur.USDT = decimal.Zero
		}
		if cur.Ispay.IsNegative() {
			cur.Ispay = decimal.Zero
		}
		if cur.Value.IsNegative() || (!cur.USDT.IsPositive() && !cur.Ispay.IsPositive()) {
			cur.Value = decimal.Zero
		}
		if !cur.Value.IsPositive() && !cur.USDT.IsPositive() && !cur.Ispay.IsPositive() {
			t := now
			cur.BurnedAt = &t
			cur.Value = decimal.Zero
			cur.USDT = decimal.Zero
			cur.Ispay = decimal.Zero
		}
		if err := uc.daily.SaveHold(ctx, cur); err != nil {
			return err
		}
		remainU = money.Round(remainU.Sub(takeU))
		remainI = money.Round(remainI.Sub(takeI))
	}
	return nil
}
