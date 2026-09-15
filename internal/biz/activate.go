package biz

import (
	"context"
	"time"

	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
)

const (
	remarkActivateUSDT   = "activate move lock to available"
	remarkActivateIspay  = "activate move lock ispay"
	remarkDailyLockUSDT  = "daily lock release"
	remarkDailyLockIspay = "daily lock release ispay"
)

// orderActivateCap 购买单对应的封顶，用于激活时解冻额度。
func orderActivateCap(_ context.Context, _ PackageRepo, amount decimal.Decimal) decimal.Decimal {
	return CapForAmount(amount)
}

func activateMoveUSDT(lock, cap decimal.Decimal) decimal.Decimal {
	lock = money.Round(lock)
	cap = money.Round(cap)
	if !lock.IsPositive() || !cap.IsPositive() {
		return decimal.Zero
	}
	if lock.LessThanOrEqual(cap) {
		return lock
	}
	return cap
}

func activateMoveIspay(lockUSDT, moveUSDT, lockIspay decimal.Decimal) decimal.Decimal {
	lockUSDT = money.Round(lockUSDT)
	moveUSDT = money.Round(moveUSDT)
	lockIspay = money.Round(lockIspay)
	if !lockIspay.IsPositive() || !moveUSDT.IsPositive() || !lockUSDT.IsPositive() {
		return decimal.Zero
	}
	return money.Round(lockIspay.Mul(moveUSDT).Div(lockUSDT))
}

// ApplyPaidOrderCap 支付后立刻按本单金额抬升 cap_effective（取更大档，不降档）。
func ApplyPaidOrderCap(ctx context.Context, users UserRepo, packages PackageRepo, userID uint64, orderAmount decimal.Decimal) error {
	if users == nil || userID == 0 {
		return nil
	}
	want := orderActivateCap(ctx, packages, orderAmount)
	u, err := users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if !want.GreaterThan(money.Round(u.CapEffective)) {
		return nil
	}
	return users.SetCapEffective(ctx, userID, want)
}

// ReleaseInactiveLock 已激活用户按给定封顶把未入超额批次的冻结转到可提现 / ispay。
func ReleaseInactiveLock(ctx context.Context, users UserRepo, balances UserBalanceRepo, ledger LedgerRepo, userID uint64, cap decimal.Decimal) error {
	return ReleaseLockByCap(ctx, users, balances, ledger, userID, cap, nil, remarkActivateUSDT, remarkActivateIspay)
}

// ReleaseLockByCap 按封顶解冻；settleDay+remark 用于日结防重。
func ReleaseLockByCap(ctx context.Context, users UserRepo, balances UserBalanceRepo, ledger LedgerRepo, userID uint64, cap decimal.Decimal, settleDay *time.Time, usdtRemark, ispayRemark string) error {
	return ReleaseLockByCapReserved(ctx, users, balances, ledger, userID, cap, decimal.Zero, decimal.Zero, settleDay, usdtRemark, ispayRemark)
}

// ReleaseLockByCapReserved 按封顶解冻，但跳过 reserved（超额冻结由买单 daily_cap 解冻或 72h 清除）。
func ReleaseLockByCapReserved(ctx context.Context, users UserRepo, balances UserBalanceRepo, ledger LedgerRepo, userID uint64, cap, reservedUSDT, reservedIspay decimal.Decimal, settleDay *time.Time, usdtRemark, ispayRemark string) error {
	if users == nil || balances == nil || ledger == nil || userID == 0 {
		return nil
	}
	if usdtRemark == "" {
		usdtRemark = remarkActivateUSDT
	}
	if ispayRemark == "" {
		ispayRemark = remarkActivateIspay
	}
	u, err := users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if !u.IsActivated() {
		return nil
	}
	lock := money.Round(u.LockBalance.Sub(money.Round(reservedUSDT)))
	lockIspay := money.Round(u.LockIspay.Sub(money.Round(reservedIspay)))
	if lock.IsNegative() {
		lock = decimal.Zero
	}
	if lockIspay.IsNegative() {
		lockIspay = decimal.Zero
	}
	move := activateMoveUSDT(lock, cap)
	if move.IsPositive() {
		if err := balances.SubLockBalance(ctx, userID, move); err != nil {
			return err
		}
		if err := balances.AddAvailableBalance(ctx, userID, move); err != nil {
			return err
		}
		if err := ledger.Create(ctx, &LedgerEntry{
			UserID: userID, EntryType: LedgerActivate, Amount: move.Neg(),
			BalanceKind: BalanceLock, SettleDate: settleDay, Remark: usdtRemark,
		}); err != nil {
			return err
		}
		if err := ledger.Create(ctx, &LedgerEntry{
			UserID: userID, EntryType: LedgerActivate, Amount: move,
			BalanceKind: BalanceAvailable, SettleDate: settleDay, Remark: usdtRemark,
		}); err != nil {
			return err
		}
	}
	moveIspay := activateMoveIspay(lock, move, lockIspay)
	if moveIspay.IsPositive() {
		if err := balances.SubLockIspay(ctx, userID, moveIspay); err != nil {
			return err
		}
		if err := balances.AddIspayBalance(ctx, userID, moveIspay); err != nil {
			return err
		}
		if err := ledger.Create(ctx, &LedgerEntry{
			UserID: userID, EntryType: LedgerActivateIspay, Amount: moveIspay.Neg(),
			BalanceKind: BalanceLockIspay, SettleDate: settleDay, Remark: ispayRemark,
		}); err != nil {
			return err
		}
		if err := ledger.Create(ctx, &LedgerEntry{
			UserID: userID, EntryType: LedgerActivateIspay, Amount: moveIspay,
			BalanceKind: BalanceIspay, SettleDate: settleDay, Remark: ispayRemark,
		}); err != nil {
			return err
		}
	}
	return nil
}

// LockUnlockPreview 剩余冻结与当日还能日结解冻的额度。
type LockUnlockPreview struct {
	LockUSDT         decimal.Decimal
	LockIspay        decimal.Decimal
	CapEffective     decimal.Decimal
	UnlockTodayUSDT  decimal.Decimal
	UnlockTodayIspay decimal.Decimal
	TodayReleased    bool
}

// ComputeLockUnlock 按当前冻结和封顶算出当日可解冻；未激活或今日已日结解冻则为 0。
func ComputeLockUnlock(user *User, todayReleased bool) LockUnlockPreview {
	p := LockUnlockPreview{TodayReleased: todayReleased}
	if user == nil {
		return p
	}
	p.LockUSDT = money.Round(user.LockBalance)
	p.LockIspay = money.Round(user.LockIspay)
	p.CapEffective = money.Round(user.CapEffective)
	if !user.IsActivated() || todayReleased {
		return p
	}
	p.UnlockTodayUSDT = activateMoveUSDT(p.LockUSDT, p.CapEffective)
	p.UnlockTodayIspay = activateMoveIspay(p.LockUSDT, p.UnlockTodayUSDT, p.LockIspay)
	return p
}

// PreviewUserLockUnlock 读今日日结解冻流水，返回剩余冻结与当日还能解冻多少。
func (uc *SettleUseCase) PreviewUserLockUnlock(ctx context.Context, user *User) (LockUnlockPreview, error) {
	if user == nil {
		return LockUnlockPreview{}, nil
	}
	released := false
	if uc != nil && uc.ledger != nil && uc.now != nil && user.IsActivated() {
		settleDay, _ := uc.settleDay(uc.now())
		ok, err := uc.ledger.ExistsByUserTypeDateRemark(ctx, user.ID, LedgerActivate, settleDay, remarkDailyLockUSDT)
		if err != nil {
			return LockUnlockPreview{}, err
		}
		released = ok
	}
	p := ComputeLockUnlock(user, released)
	if uc != nil && uc.daily != nil && user.ID != 0 {
		ou, oi, err := uc.daily.ActiveTotals(ctx, user.ID)
		if err != nil {
			return LockUnlockPreview{}, err
		}
		residual := money.Round(p.LockUSDT.Sub(ou))
		residualIspay := money.Round(p.LockIspay.Sub(oi))
		if residual.IsNegative() {
			residual = decimal.Zero
		}
		if residualIspay.IsNegative() {
			residualIspay = decimal.Zero
		}
		if !user.IsActivated() || released {
			p.UnlockTodayUSDT = decimal.Zero
			p.UnlockTodayIspay = decimal.Zero
		} else {
			p.UnlockTodayUSDT = activateMoveUSDT(residual, p.CapEffective)
			p.UnlockTodayIspay = activateMoveIspay(residual, p.UnlockTodayUSDT, residualIspay)
		}
	}
	return p, nil
}
