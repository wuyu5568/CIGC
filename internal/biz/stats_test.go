package biz

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

type memStats struct {
	total, activated, todayR, todayA int64
	buyTotal, todayBuy               decimal.Decimal
	avail, ispay                     decimal.Decimal
	todayStatic, todayDyn, totalRew  decimal.Decimal
	todayWD, totalWD                 decimal.Decimal
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
func (m *memStats) SumPaidOrders(_ context.Context, from, to *time.Time) (decimal.Decimal, error) {
	if from != nil {
		return m.todayBuy, nil
	}
	return m.buyTotal, nil
}
func (m *memStats) SumAvailableUSDT(context.Context) (decimal.Decimal, error) { return m.avail, nil }
func (m *memStats) SumIspay(context.Context) (decimal.Decimal, error)         { return m.ispay, nil }
func (m *memStats) SumLedgerUSDT(_ context.Context, types []string, from, to *time.Time) (decimal.Decimal, error) {
	if from != nil {
		for _, t := range types {
			if t == LedgerStatic {
				return m.todayStatic, nil
			}
		}
		return m.todayDyn, nil
	}
	return m.totalRew, nil
}
func (m *memStats) SumWithdrawUSDT(_ context.Context, from, to *time.Time) (decimal.Decimal, error) {
	if from != nil {
		return m.todayWD, nil
	}
	return m.totalWD, nil
}

func TestStatsDashboard(t *testing.T) {
	uc := NewStatsUseCase(&memStats{
		total: 10, activated: 4, todayR: 2, todayA: 1,
		buyTotal: decimal.RequireFromString("5000"), todayBuy: decimal.RequireFromString("1000"),
		avail: decimal.RequireFromString("200"), ispay: decimal.RequireFromString("3"),
		todayStatic: decimal.RequireFromString("10"), todayDyn: decimal.RequireFromString("20"),
		totalRew: decimal.RequireFromString("300"),
		todayWD:  decimal.RequireFromString("50"), totalWD: decimal.RequireFromString("80"),
	})
	uc.now = func() time.Time { return time.Date(2026, 9, 13, 12, 0, 0, 0, shanghaiLoc()) }
	got, err := uc.Dashboard(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalUserR != 10 || got.TotalUser != 4 || got.TodayUserR != 2 || got.TodayUser != 1 {
		t.Fatalf("users %+v", got)
	}
	if !got.TodayThree.Equal(decimal.RequireFromString("30")) {
		t.Fatalf("todayThree=%s", got.TodayThree)
	}
	if !got.BuyTotal.Equal(decimal.RequireFromString("5000")) || !got.TodayWithdraw.Equal(decimal.RequireFromString("50")) {
		t.Fatalf("money %+v", got)
	}
}
