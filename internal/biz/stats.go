package biz

import (
	"context"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// DashboardStats 管理端首页 /all 看板。
type DashboardStats struct {
	TotalUserR         int64
	TotalUser          int64
	TodayUserR         int64
	TodayUser          int64
	OrderCount         int64
	TodayOrderCount    int64
	BuyTotal           decimal.Decimal
	TodayBuy           decimal.Decimal
	DepositTotal       decimal.Decimal
	TodayDeposit       decimal.Decimal
	RechargeRemain     decimal.Decimal
	AdminRechargeNet   decimal.Decimal
	BalanceUSDT        decimal.Decimal
	BalanceIspay       decimal.Decimal
	LockUSDT           decimal.Decimal
	LockIspay          decimal.Decimal
	FrozenUSDT         decimal.Decimal
	FrozenIspay        decimal.Decimal
	TodayOne           decimal.Decimal
	TodayTwo           decimal.Decimal
	TodayIspayDyn      decimal.Decimal
	TodayThree         decimal.Decimal
	TotalStatic        decimal.Decimal
	TotalDynamic       decimal.Decimal
	TotalReward        decimal.Decimal
	TodayWithdraw      decimal.Decimal
	TotalWithdraw      decimal.Decimal
	TodayWithdrawIspay decimal.Decimal
	TotalWithdrawIspay decimal.Decimal
}

// StatsRepo 管理端汇总查询。
type StatsRepo interface {
	CountUsers(ctx context.Context) (total, activated int64, err error)
	CountUsersCreatedBetween(ctx context.Context, from, to time.Time) (int64, error)
	CountFirstActivatedBetween(ctx context.Context, from, to time.Time) (int64, error)
	CountPaidOrders(ctx context.Context, from, to *time.Time) (int64, error)
	SumPaidOrders(ctx context.Context, from, to *time.Time) (decimal.Decimal, error)
	SumMatchedDeposits(ctx context.Context, from, to *time.Time) (decimal.Decimal, error)
	SumRechargeBalance(ctx context.Context) (decimal.Decimal, error)
	SumAvailableUSDT(ctx context.Context) (decimal.Decimal, error)
	SumIspay(ctx context.Context) (decimal.Decimal, error)
	SumLockUSDT(ctx context.Context) (decimal.Decimal, error)
	SumLockIspay(ctx context.Context) (decimal.Decimal, error)
	SumFrozenUSDT(ctx context.Context) (decimal.Decimal, error)
	SumFrozenIspay(ctx context.Context) (decimal.Decimal, error)
	SumLedgerUSDT(ctx context.Context, types []string, from, to *time.Time) (decimal.Decimal, error)
	SumLedgerByKinds(ctx context.Context, types, kinds []string, from, to *time.Time, positiveOnly bool) (decimal.Decimal, error)
	SumWithdrawCredited(ctx context.Context, asset string, from, to *time.Time) (decimal.Decimal, error)
}

// StatsUseCase 管理端首页统计。
type StatsUseCase struct {
	stats StatsRepo
	now   func() time.Time
	loc   *time.Location
}

// NewStatsUseCase 构造统计用例。
func NewStatsUseCase(stats StatsRepo) *StatsUseCase {
	return &StatsUseCase{stats: stats, now: time.Now, loc: shanghaiLoc()}
}

// SetTimezone 日切时区，空则上海。
func (uc *StatsUseCase) SetTimezone(tz string) {
	if uc == nil {
		return
	}
	if strings.TrimSpace(tz) == "" {
		uc.loc = shanghaiLoc()
		return
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		uc.loc = shanghaiLoc()
		return
	}
	uc.loc = loc
}

func (uc *StatsUseCase) dayRange(now time.Time) (time.Time, time.Time) {
	loc := uc.loc
	if loc == nil {
		loc = shanghaiLoc()
	}
	t := now.In(loc)
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	return start, start.Add(24 * time.Hour)
}

type dashAcc struct{ err error }

func (a *dashAcc) i64(v int64, err error) int64 {
	if a.err == nil {
		a.err = err
	}
	return v
}

func (a *dashAcc) dec(v decimal.Decimal, err error) decimal.Decimal {
	if a.err == nil {
		a.err = err
	}
	return v
}

// Dashboard 聚合首页指标。
func (uc *StatsUseCase) Dashboard(ctx context.Context) (*DashboardStats, error) {
	out := &DashboardStats{}
	if uc == nil || uc.stats == nil {
		return out, nil
	}
	from, to := uc.dayRange(uc.now())
	var acc dashAcc

	total, activated, err := uc.stats.CountUsers(ctx)
	if err != nil {
		return nil, err
	}
	out.TotalUserR = total
	out.TotalUser = activated
	out.TodayUserR = acc.i64(uc.stats.CountUsersCreatedBetween(ctx, from, to))
	out.TodayUser = acc.i64(uc.stats.CountFirstActivatedBetween(ctx, from, to))
	out.OrderCount = acc.i64(uc.stats.CountPaidOrders(ctx, nil, nil))
	out.TodayOrderCount = acc.i64(uc.stats.CountPaidOrders(ctx, &from, &to))
	out.BuyTotal = acc.dec(uc.stats.SumPaidOrders(ctx, nil, nil))
	out.TodayBuy = acc.dec(uc.stats.SumPaidOrders(ctx, &from, &to))
	out.DepositTotal = acc.dec(uc.stats.SumMatchedDeposits(ctx, nil, nil))
	out.TodayDeposit = acc.dec(uc.stats.SumMatchedDeposits(ctx, &from, &to))
	out.RechargeRemain = acc.dec(uc.stats.SumRechargeBalance(ctx))
	out.AdminRechargeNet = acc.dec(uc.stats.SumLedgerByKinds(ctx, []string{LedgerAdminAdjust}, []string{BalanceRecharge}, nil, nil, false))
	out.BalanceUSDT = acc.dec(uc.stats.SumAvailableUSDT(ctx))
	out.BalanceIspay = acc.dec(uc.stats.SumIspay(ctx))
	out.LockUSDT = acc.dec(uc.stats.SumLockUSDT(ctx))
	out.LockIspay = acc.dec(uc.stats.SumLockIspay(ctx))
	out.FrozenUSDT = acc.dec(uc.stats.SumFrozenUSDT(ctx))
	out.FrozenIspay = acc.dec(uc.stats.SumFrozenIspay(ctx))

	staticTypes := []string{LedgerStatic}
	dynamicTypes := []string{LedgerDirect, LedgerMatch, LedgerManage}
	ispayDynTypes := []string{LedgerDirectIspay, LedgerMatchIspay, LedgerManageIspay}
	ispayKinds := []string{BalanceIspay, BalanceLockIspay}

	todayStatic := acc.dec(uc.stats.SumLedgerUSDT(ctx, staticTypes, &from, &to))
	todayDyn := acc.dec(uc.stats.SumLedgerUSDT(ctx, dynamicTypes, &from, &to))
	out.TodayIspayDyn = acc.dec(uc.stats.SumLedgerByKinds(ctx, ispayDynTypes, ispayKinds, &from, &to, true))
	totalStatic := acc.dec(uc.stats.SumLedgerUSDT(ctx, staticTypes, nil, nil))
	totalDyn := acc.dec(uc.stats.SumLedgerUSDT(ctx, dynamicTypes, nil, nil))
	out.TodayOne = ValueFromUSDTHalf(todayStatic)
	out.TodayTwo = ValueFromUSDTHalf(todayDyn)
	out.TodayThree = out.TodayOne.Add(out.TodayTwo)
	out.TotalStatic = ValueFromUSDTHalf(totalStatic)
	out.TotalDynamic = ValueFromUSDTHalf(totalDyn)
	out.TotalReward = out.TotalStatic.Add(out.TotalDynamic)

	out.TodayWithdraw = acc.dec(uc.stats.SumWithdrawCredited(ctx, WithdrawAssetUSDT, &from, &to))
	out.TotalWithdraw = acc.dec(uc.stats.SumWithdrawCredited(ctx, WithdrawAssetUSDT, nil, nil))
	out.TodayWithdrawIspay = acc.dec(uc.stats.SumWithdrawCredited(ctx, WithdrawAssetIspay, &from, &to))
	out.TotalWithdrawIspay = acc.dec(uc.stats.SumWithdrawCredited(ctx, WithdrawAssetIspay, nil, nil))
	if acc.err != nil {
		return nil, acc.err
	}
	return out, nil
}
