package biz

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestStaticDaily_300And750(t *testing.T) {
	spot := decimal.RequireFromString("2000")
	coins, daily, value, usdt, ispay, ok := StaticDaily(decimal.RequireFromString("12000"), 300, spot)
	if !ok {
		t.Fatal("300 should be ok")
	}
	if !coins.Equal(decimal.RequireFromString("10")) {
		t.Fatalf("coins=%s", coins)
	}
	if !daily.Equal(decimal.RequireFromString("0.03333333")) {
		t.Fatalf("daily=%s", daily)
	}
	if !value.Equal(decimal.RequireFromString("66.66666000")) {
		t.Fatalf("value=%s", value)
	}
	if !usdt.Equal(decimal.RequireFromString("33.33333000")) || !ispay.Equal(decimal.RequireFromString("0.01666667")) {
		t.Fatalf("usdt=%s ispay=%s", usdt, ispay)
	}

	coins, daily, value, usdt, ispay, ok = StaticDaily(decimal.RequireFromString("12000"), 750, spot)
	if !ok || !coins.Equal(decimal.RequireFromString("15")) || !daily.Equal(decimal.RequireFromString("0.02000000")) {
		t.Fatalf("750 coins=%s daily=%s ok=%v", coins, daily, ok)
	}
	if !value.Equal(decimal.RequireFromString("40")) || !usdt.Equal(decimal.RequireFromString("20")) || !ispay.Equal(decimal.RequireFromString("0.01000000")) {
		t.Fatalf("750 value=%s usdt=%s ispay=%s", value, usdt, ispay)
	}
}

func TestSplitHalfIspay(t *testing.T) {
	usdt, ispay := SplitHalfIspay(decimal.RequireFromString("40"), decimal.RequireFromString("2000"))
	if !usdt.Equal(decimal.RequireFromString("20")) || !ispay.Equal(decimal.RequireFromString("0.01")) {
		t.Fatalf("usdt=%s ispay=%s", usdt, ispay)
	}
}

func TestReleaseBuyPrice(t *testing.T) {
	if _, ok := ReleaseBuyPrice(90); ok {
		t.Fatal("invalid days")
	}
	p, ok := ReleaseBuyPrice(600)
	if !ok || !p.Equal(decimal.NewFromInt(1000)) {
		t.Fatalf("%s %v", p, ok)
	}
}

func TestComputeOrderStaticRelease_NoHistory(t *testing.T) {
	spot := decimal.RequireFromString("2000")
	o := &Order{Status: OrderPaid, Amount: decimal.RequireFromString("12000"), ReleaseDays: 300}
	r := ComputeOrderStaticRelease(o, spot, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, 0)
	if !r.TodayUSDT.Equal(decimal.RequireFromString("33.33333000")) || !r.TodayIspay.Equal(decimal.RequireFromString("0.01666667")) {
		t.Fatalf("today usdt=%s ispay=%s", r.TodayUSDT, r.TodayIspay)
	}
	if !r.ReleasedUSDT.IsZero() || !r.ReleasedIspay.IsZero() {
		t.Fatalf("released usdt=%s ispay=%s", r.ReleasedUSDT, r.ReleasedIspay)
	}
	if !r.PendingUSDT.Equal(decimal.RequireFromString("10000")) || !r.PendingIspay.Equal(decimal.RequireFromString("5")) {
		t.Fatalf("pending usdt=%s ispay=%s", r.PendingUSDT, r.PendingIspay)
	}
}

func TestComputeOrderStaticRelease_OneDayReleased(t *testing.T) {
	spot := decimal.RequireFromString("2000")
	o := &Order{Status: OrderPaid, Amount: decimal.RequireFromString("12000"), ReleaseDays: 300}
	todayU := decimal.RequireFromString("33.33333000")
	todayI := decimal.RequireFromString("0.01666667")
	r := ComputeOrderStaticRelease(o, spot, todayU, todayI, todayU, todayI, 1)
	if !r.TodayUSDT.Equal(todayU) || !r.TodayIspay.Equal(todayI) {
		t.Fatalf("today usdt=%s ispay=%s", r.TodayUSDT, r.TodayIspay)
	}
	if !r.ReleasedUSDT.Equal(todayU) || !r.ReleasedIspay.Equal(todayI) {
		t.Fatalf("released usdt=%s ispay=%s", r.ReleasedUSDT, r.ReleasedIspay)
	}
	releasedValue := todayU.Add(todayI.Mul(spot))
	pendingValue := decimal.RequireFromString("20000").Sub(releasedValue)
	wantU, wantI := SplitHalfIspay(pendingValue, spot)
	if !r.PendingUSDT.Equal(wantU) || !r.PendingIspay.Equal(wantI) {
		t.Fatalf("pending usdt=%s/%s ispay=%s/%s", r.PendingUSDT, wantU, r.PendingIspay, wantI)
	}
}

func TestComputeOrderStaticRelease_Finished(t *testing.T) {
	spot := decimal.RequireFromString("2000")
	o := &Order{Status: OrderPaid, Amount: decimal.RequireFromString("12000"), ReleaseDays: 300}
	r := ComputeOrderStaticRelease(o, spot, decimal.Zero, decimal.Zero, decimal.RequireFromString("10000"), decimal.RequireFromString("5"), 300)
	if !r.TodayUSDT.IsZero() || !r.TodayIspay.IsZero() {
		t.Fatalf("today should be 0: %s %s", r.TodayUSDT, r.TodayIspay)
	}
	if !r.PendingUSDT.IsZero() || !r.PendingIspay.IsZero() {
		t.Fatalf("pending should be 0: %s %s", r.PendingUSDT, r.PendingIspay)
	}
}
