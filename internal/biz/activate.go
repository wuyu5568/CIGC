package biz

import (
	"context"
	"strings"
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
		settleDay := uc.currentBusinessDay(ctx)
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

func overflowHoldName(source string) string {
	switch source {
	case LedgerDirect:
		return "直推冻结"
	case LedgerMatch:
		return "对碰冻结"
	case LedgerManage:
		return "管理冻结"
	case LedgerStatic:
		return "静态冻结"
	case overflowSourceInactive:
		return "未激活冻结"
	case overflowSourceAdmin:
		return "调账冻结"
	default:
		return "冻结"
	}
}

// ListUserFreezeAssets 用户端冻结资产：待日结解冻汇总 + 批次剩余。
func (uc *SettleUseCase) ListUserFreezeAssets(ctx context.Context, userID uint64) ([]*RewardItem, error) {
	if uc == nil || userID == 0 {
		return []*RewardItem{}, nil
	}
	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	holds, err := uc.daily.ListActiveHolds(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := freezeAssetItems(user, holds, uc.location())
	if err := NewLedgerUseCase(uc.ledger, uc.orders, uc.users).attachOrderSources(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListAdminFreezeAssets 管理端冻结资产明细：待日结解冻 + 批次剩余，可按地址模糊筛。
func (uc *SettleUseCase) ListAdminFreezeAssets(ctx context.Context, address string, page int) (*RewardPage, error) {
	if uc == nil || uc.users == nil {
		return &RewardPage{Items: []*RewardItem{}}, nil
	}
	users, err := uc.users.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	var holds []*CapOverflowHold
	if uc.daily != nil {
		holds, err = uc.daily.ListActiveHoldsAll(ctx)
		if err != nil {
			return nil, err
		}
	}
	byUser := map[uint64][]*CapOverflowHold{}
	for _, h := range holds {
		if h == nil {
			continue
		}
		byUser[h.UserID] = append(byUser[h.UserID], h)
	}
	q := strings.ToLower(strings.TrimSpace(address))
	loc := uc.location()
	out := make([]*RewardItem, 0)
	for _, u := range users {
		if u == nil {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(u.Address), q) {
			continue
		}
		rows := freezeAssetItems(u, byUser[u.ID], loc)
		if len(rows) == 0 {
			continue
		}
		for _, it := range rows {
			if it == nil {
				continue
			}
			it.Address = u.Address
			it.Category = "冻结资产"
		}
		out = append(out, rows...)
	}
	if err := NewLedgerUseCase(uc.ledger, uc.orders, uc.users).attachOrderSources(ctx, out); err != nil {
		return nil, err
	}
	return &RewardPage{
		Items: paginateRewards(out, page, DefaultRewardPageSize),
		Total: len(out),
	}, nil
}

func freezeAssetItems(user *User, holds []*CapOverflowHold, loc *time.Location) []*RewardItem {
	if user == nil {
		return []*RewardItem{}
	}
	holdUSDT := decimal.Zero
	holdIspay := decimal.Zero
	holdItems := make([]*RewardItem, 0, len(holds))
	if loc == nil {
		loc = time.UTC
	}
	for _, h := range holds {
		if h == nil {
			continue
		}
		holdUSDT = holdUSDT.Add(h.USDT)
		holdIspay = holdIspay.Add(h.Ispay)
		item := &RewardItem{
			ID:          h.ID,
			Name:        overflowHoldName(h.SourceType),
			Reason:      "freeze_hold",
			Category:    "冻结资产",
			Amount:      amountStr(h.USDT),
			AmountTwo:   amountStr(h.Ispay),
			OrderID:     orderIDStr(h.OrderID),
			BalanceKind: BalanceLock,
			BalanceName: BalanceKindName(BalanceLock),
			CreatedAt:   h.CreatedAt,
		}
		if h.ExpiresAt != nil {
			item.SettleDate = "到期 " + h.ExpiresAt.In(loc).Format("2006-01-02 15:04")
		} else {
			item.SettleDate = "待封账"
		}
		holdItems = append(holdItems, item)
	}
	pendingUSDT := money.Round(user.LockBalance.Sub(holdUSDT))
	pendingIspay := money.Round(user.LockIspay.Sub(holdIspay))
	if pendingUSDT.IsNegative() {
		pendingUSDT = decimal.Zero
	}
	if pendingIspay.IsNegative() {
		pendingIspay = decimal.Zero
	}
	out := make([]*RewardItem, 0, 1+len(holdItems))
	if pendingUSDT.IsPositive() || pendingIspay.IsPositive() {
		out = append(out, &RewardItem{
			Name:        "待日结解冻",
			Detail:      "非批次冻结，按封顶排队",
			Reason:      "freeze_pending",
			Category:    "冻结资产",
			Amount:      amountStr(pendingUSDT),
			AmountTwo:   amountStr(pendingIspay),
			BalanceKind: BalanceLock,
			BalanceName: BalanceKindName(BalanceLock),
		})
	}
	out = append(out, holdItems...)
	return out
}
