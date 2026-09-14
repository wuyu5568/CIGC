package biz

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

type memStats struct {
	total, activated, todayR, todayA  int64
	orderCount, todayOrderCount       int64
	buyTotal, todayBuy                decimal.Decimal
	depositTotal, todayDeposit        decimal.Decimal
	rechargeRemain, adminRecharge     decimal.Decimal
	avail, ispay                      decimal.Decimal
	lockUSDT, lockIspay               decimal.Decimal
	frozenUSDT, frozenIspay           decimal.Decimal
	todayStatic, todayDyn, todayIspay decimal.Decimal
	totalStatic, totalDyn             decimal.Decimal
	todayWD, totalWD                  decimal.Decimal
	todayWDIspay, totalWDIspay        decimal.Decimal
}

func (m *memStats) CountUsers(context.Context) (int64, int64, error) {
	return m.total, m.activated, nil
}
func (m *memStats) CountUsersCreatedBetween(context.Context, time.Time, time.Time) (int64, error) {
	return m.todayR, nil
}
func (m *memStats) CountFirstActivatedBetween(context.Context, time.Time, time.Time) (int64, error) {
	return m.todayA, nil
}
func (m *memStats) CountPaidOrders(_ context.Context, from, to *time.Time) (int64, error) {
	if from != nil {
		return m.todayOrderCount, nil
	}
	return m.orderCount, nil
}
func (m *memStats) SumPaidOrders(_ context.Context, from, to *time.Time) (decimal.Decimal, error) {
	if from != nil {
		return m.todayBuy, nil
	}
	return m.buyTotal, nil
}
func (m *memStats) SumMatchedDeposits(_ context.Context, from, to *time.Time) (decimal.Decimal, error) {
	if from != nil {
		return m.todayDeposit, nil
	}
	return m.depositTotal, nil
}
func (m *memStats) SumRechargeBalance(context.Context) (decimal.Decimal, error) {
	return m.rechargeRemain, nil
}
func (m *memStats) SumAvailableUSDT(context.Context) (decimal.Decimal, error) { return m.avail, nil }
func (m *memStats) SumIspay(context.Context) (decimal.Decimal, error)         { return m.ispay, nil }
func (m *memStats) SumLockUSDT(context.Context) (decimal.Decimal, error)      { return m.lockUSDT, nil }
func (m *memStats) SumLockIspay(context.Context) (decimal.Decimal, error)     { return m.lockIspay, nil }
func (m *memStats) SumFrozenUSDT(context.Context) (decimal.Decimal, error)    { return m.frozenUSDT, nil }
func (m *memStats) SumFrozenIspay(context.Context) (decimal.Decimal, error) {
	return m.frozenIspay, nil
}
func (m *memStats) SumLedgerUSDT(_ context.Context, types []string, from, to *time.Time) (decimal.Decimal, error) {
	static := false
	for _, t := range types {
		if t == LedgerStatic {
			static = true
			break
		}
	}
	if from != nil {
		if static {
			return m.todayStatic, nil
		}
		return m.todayDyn, nil
	}
	if static {
		return m.totalStatic, nil
	}
	return m.totalDyn, nil
}
func (m *memStats) SumLedgerByKinds(_ context.Context, types, _ []string, from, to *time.Time, _ bool) (decimal.Decimal, error) {
	for _, t := range types {
		if t == LedgerAdminAdjust {
			return m.adminRecharge, nil
		}
		if t == LedgerDirectIspay || t == LedgerMatchIspay || t == LedgerManageIspay {
			if from != nil {
				return m.todayIspay, nil
			}
			return decimal.Zero, nil
		}
	}
	return decimal.Zero, nil
}
func (m *memStats) SumWithdrawCredited(_ context.Context, asset string, from, to *time.Time) (decimal.Decimal, error) {
	if asset == WithdrawAssetIspay {
		if from != nil {
			return m.todayWDIspay, nil
		}
		return m.totalWDIspay, nil
	}
	if from != nil {
		return m.todayWD, nil
	}
	return m.totalWD, nil
}

func TestStatsDashboard(t *testing.T) {
	uc := NewStatsUseCase(&memStats{
		total: 10, activated: 4, todayR: 2, todayA: 1,
		orderCount: 7, todayOrderCount: 3,
		buyTotal: decimal.RequireFromString("5000"), todayBuy: decimal.RequireFromString("1000"),
		depositTotal: decimal.RequireFromString("8000"), todayDeposit: decimal.RequireFromString("200"),
		rechargeRemain: decimal.RequireFromString("150"), adminRecharge: decimal.RequireFromString("50"),
		avail: decimal.RequireFromString("200"), ispay: decimal.RequireFromString("3"),
		lockUSDT: decimal.RequireFromString("40"), lockIspay: decimal.RequireFromString("0.02"),
		frozenUSDT: decimal.RequireFromString("10"), frozenIspay: decimal.RequireFromString("0.01"),
		todayStatic: decimal.RequireFromString("10"), todayDyn: decimal.RequireFromString("20"),
		todayIspay:  decimal.RequireFromString("0.01"),
		totalStatic: decimal.RequireFromString("100"), totalDyn: decimal.RequireFromString("200"),
		todayWD: decimal.RequireFromString("18"), totalWD: decimal.RequireFromString("72"),
		todayWDIspay: decimal.RequireFromString("0"), totalWDIspay: decimal.RequireFromString("0"),
	})
	uc.now = func() time.Time { return time.Date(2026, 9, 13, 12, 0, 0, 0, shanghaiLoc()) }
	got, err := uc.Dashboard(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalUserR != 10 || got.TotalUser != 4 || got.TodayUserR != 2 || got.TodayUser != 1 {
		t.Fatalf("users %+v", got)
	}
	if got.OrderCount != 7 || got.TodayOrderCount != 3 {
		t.Fatalf("orders %+v", got)
	}
	if !got.TodayOne.Equal(decimal.RequireFromString("20")) || !got.TodayTwo.Equal(decimal.RequireFromString("40")) {
		t.Fatalf("today value one=%s two=%s", got.TodayOne, got.TodayTwo)
	}
	if !got.TodayThree.Equal(decimal.RequireFromString("60")) {
		t.Fatalf("todayThree=%s", got.TodayThree)
	}
	if !got.TotalStatic.Equal(decimal.RequireFromString("200")) || !got.TotalDynamic.Equal(decimal.RequireFromString("400")) {
		t.Fatalf("totals static=%s dyn=%s", got.TotalStatic, got.TotalDynamic)
	}
	if !got.AdminRechargeNet.Equal(decimal.RequireFromString("50")) || !got.DepositTotal.Equal(decimal.RequireFromString("8000")) {
		t.Fatalf("recharge %+v", got)
	}
	if !got.TodayWithdraw.Equal(decimal.RequireFromString("18")) || !got.BalanceIspay.Equal(decimal.RequireFromString("3")) {
		t.Fatalf("money %+v", got)
	}
}
