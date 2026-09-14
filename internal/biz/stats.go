package biz

import (
	"context"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// DashboardStats 管理端首页 /all 看板。
type DashboardStats struct {
	TotalUserR    int64
	TotalUser     int64
	TodayUserR    int64
	TodayUser     int64
	BuyTotal      decimal.Decimal
	TodayBuy      decimal.Decimal
	BalanceUSDT   decimal.Decimal
	TodayOne      decimal.Decimal
	TodayTwo      decimal.Decimal
	TodayThree    decimal.Decimal
	TotalReward   decimal.Decimal
	TodayWithdraw decimal.Decimal
	TotalWithdraw decimal.Decimal
	TotalIspay    decimal.Decimal
}

// StatsRepo 管理端汇总查询。
type StatsRepo interface {
	CountUsers(ctx context.Context) (total, activated int64, err error)
	CountUsersCreatedBetween(ctx context.Context, from, to time.Time) (int64, error)
	CountFirstActivatedBetween(ctx context.Context, from, to time.Time) (int64, error)
	SumPaidOrders(ctx context.Context, from, to *time.Time) (decimal.Decimal, error)
	SumAvailableUSDT(ctx context.Context) (decimal.Decimal, error)
	SumIspay(ctx context.Context) (decimal.Decimal, error)
	SumLedgerUSDT(ctx context.Context, types []string, from, to *time.Time) (decimal.Decimal, error)
	SumWithdrawUSDT(ctx context.Context, from, to *time.Time) (decimal.Decimal, error)
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

// Dashboard 聚合首页指标。
func (uc *StatsUseCase) Dashboard(ctx context.Context) (*DashboardStats, error) {
	out := &DashboardStats{}
	if uc == nil || uc.stats == nil {
		return out, nil
	}
	from, to := uc.dayRange(uc.now())

	total, activated, err := uc.stats.CountUsers(ctx)
	if err != nil {
		return nil, err
	}
	out.TotalUserR = total
	out.TotalUser = activated

	out.TodayUserR, err = uc.stats.CountUsersCreatedBetween(ctx, from, to)
	if err != nil {
		return nil, err
	}
	out.TodayUser, err = uc.stats.CountFirstActivatedBetween(ctx, from, to)
	if err != nil {
		return nil, err
	}

	out.BuyTotal, err = uc.stats.SumPaidOrders(ctx, nil, nil)
	if err != nil {
		return nil, err
	}
	out.TodayBuy, err = uc.stats.SumPaidOrders(ctx, &from, &to)
	if err != nil {
		return nil, err
	}

	out.BalanceUSDT, err = uc.stats.SumAvailableUSDT(ctx)
	if err != nil {
		return nil, err
	}
	out.TotalIspay, err = uc.stats.SumIspay(ctx)
	if err != nil {
		return nil, err
	}

	staticTypes := []string{LedgerStatic}
	dynamicTypes := []string{LedgerDirect, LedgerMatch, LedgerManage}
	out.TodayOne, err = uc.stats.SumLedgerUSDT(ctx, staticTypes, &from, &to)
	if err != nil {
		return nil, err
	}
	out.TodayTwo, err = uc.stats.SumLedgerUSDT(ctx, dynamicTypes, &from, &to)
	if err != nil {
		return nil, err
	}
	out.TodayThree = out.TodayOne.Add(out.TodayTwo)
	allReward := append(append([]string{}, staticTypes...), dynamicTypes...)
	out.TotalReward, err = uc.stats.SumLedgerUSDT(ctx, allReward, nil, nil)
	if err != nil {
		return nil, err
	}

	out.TodayWithdraw, err = uc.stats.SumWithdrawUSDT(ctx, &from, &to)
	if err != nil {
		return nil, err
	}
	out.TotalWithdraw, err = uc.stats.SumWithdrawUSDT(ctx, nil, nil)
	if err != nil {
		return nil, err
	}
	return out, nil
}
