package biz

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestReleaseInactiveLock_NoopUntilPaid(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xlock"})
	if err != nil {
		t.Fatal(err)
	}
	if err := users.AddLockBalance(context.Background(), u.ID, decimal.RequireFromString("10")); err != nil {
		t.Fatal(err)
	}
	if err := users.AddLockIspay(context.Background(), u.ID, decimal.RequireFromString("1")); err != nil {
		t.Fatal(err)
	}
	led := &memLedger{}
	if err := ReleaseInactiveLock(context.Background(), users, users, led, u.ID, decimal.RequireFromString("600")); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("10")) || !got.AvailableBalance.IsZero() {
		t.Fatalf("inactive lock=%s avail=%s", got.LockBalance, got.AvailableBalance)
	}
}

func TestReleaseInactiveLock_MovesOnFirstPay(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xact"})
	if err != nil {
		t.Fatal(err)
	}
	if err := users.AddLockBalance(context.Background(), u.ID, decimal.RequireFromString("10")); err != nil {
		t.Fatal(err)
	}
	if err := users.AddLockIspay(context.Background(), u.ID, decimal.RequireFromString("1")); err != nil {
		t.Fatal(err)
	}
	led := &memLedger{}
	uc := NewOrderUseCase(&memPackages{rows: []*Package{{
		ID: 1, Amount: decimal.RequireFromString("1000"), DailyCap: decimal.RequireFromString("600"), Title: "t", Enabled: true,
	}}}, newMemOrders(users), users, users, led)
	o, err := uc.CreateOrder(context.Background(), u.ID, decimal.RequireFromString("1000"), 300)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.MarkPaid(context.Background(), o.ID); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsActivated() {
		t.Fatal("should activate after first paid order")
	}
	if !got.LockBalance.IsZero() || !got.LockIspay.IsZero() {
		t.Fatalf("lock remain usdt=%s ispay=%s", got.LockBalance, got.LockIspay)
	}
	if !got.AvailableBalance.Equal(decimal.RequireFromString("10")) || !got.IspayBalance.Equal(decimal.RequireFromString("1")) {
		t.Fatalf("avail=%s ispay=%s", got.AvailableBalance, got.IspayBalance)
	}
	if !ledgerHasType(led, LedgerActivate) || !ledgerHasType(led, LedgerActivateIspay) {
		t.Fatalf("ledger=%+v", led.rows)
	}
}

func TestSettle_InactiveCreditThenActivate(t *testing.T) {
	users := newMemUsers()
	inv, err := users.Create(context.Background(), &User{Address: "0xinv2"})
	if err != nil {
		t.Fatal(err)
	}
	invID := inv.ID
	buyer, err := users.Create(context.Background(), &User{Address: "0xbuy2", InviterID: &invID})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, buyer.ID, "3000", "2026-09-11")
	led := &memLedger{}
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, &memConfigs{}, true, shanghaiNoon("2026-09-11"))
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(context.Background(), inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("150")) {
		t.Fatalf("lock=%s", got.LockBalance)
	}

	ouc := NewOrderUseCase(&memPackages{rows: []*Package{{
		ID: 1, Amount: decimal.RequireFromString("1000"), DailyCap: decimal.RequireFromString("600"), Title: "t", Enabled: true,
	}}}, orders, users, users, led)
	ouc.SetPaidHook(uc)
	o, err := ouc.CreateOrder(context.Background(), inv.ID, decimal.RequireFromString("1000"), 300)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ouc.MarkPaid(context.Background(), o.ID); err != nil {
		t.Fatal(err)
	}
	got, err = users.FindByID(context.Background(), inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.IsZero() || !got.AvailableBalance.Equal(decimal.RequireFromString("150")) {
		t.Fatalf("after pay lock=%s avail=%s", got.LockBalance, got.AvailableBalance)
	}
}

func TestReleaseInactiveLock_CappedByOrder(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xcap"})
	if err != nil {
		t.Fatal(err)
	}
	if err := users.AddLockBalance(context.Background(), u.ID, decimal.RequireFromString("1000")); err != nil {
		t.Fatal(err)
	}
	if err := users.AddLockIspay(context.Background(), u.ID, decimal.RequireFromString("1")); err != nil {
		t.Fatal(err)
	}
	led := &memLedger{}
	uc := NewOrderUseCase(&memPackages{rows: []*Package{{
		ID: 1, Amount: decimal.RequireFromString("1000"), DailyCap: decimal.RequireFromString("600"), Title: "t", Enabled: true,
	}}}, newMemOrders(users), users, users, led)
	o, err := uc.CreateOrder(context.Background(), u.ID, decimal.RequireFromString("1000"), 300)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.MarkPaid(context.Background(), o.ID); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.AvailableBalance.Equal(decimal.RequireFromString("600")) || !got.LockBalance.Equal(decimal.RequireFromString("400")) {
		t.Fatalf("avail=%s lock=%s", got.AvailableBalance, got.LockBalance)
	}
	if !got.IspayBalance.Equal(decimal.RequireFromString("0.6")) || !got.LockIspay.Equal(decimal.RequireFromString("0.4")) {
		t.Fatalf("ispay=%s lock_ispay=%s", got.IspayBalance, got.LockIspay)
	}
	if !got.CapEffective.Equal(decimal.RequireFromString("600")) {
		t.Fatalf("cap=%s", got.CapEffective)
	}
}

func TestSettle_InactiveLockFollowsOverflowOnFirstPay(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
	if err != nil {
		t.Fatal(err)
	}
	if err := users.AddLockBalance(context.Background(), u.ID, decimal.RequireFromString("2000")); err != nil {
		t.Fatal(err)
	}
	if err := users.AddLockIspay(context.Background(), u.ID, decimal.RequireFromString("2")); err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	led := &memLedger{}
	pkgs := &memPackages{rows: seedCapPackages()}
	uc := newSettleUCFull(users, pkgs, &memSettleRuns{}, orders, led, &memConfigs{min: "10"}, true, shanghaiNoon("2026-09-11"))
	ouc := NewOrderUseCase(pkgs, orders, users, users, led)
	ouc.SetPaidHook(uc)
	o, err := ouc.CreateOrder(context.Background(), u.ID, decimal.RequireFromString("1000"), 300)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ouc.MarkPaid(context.Background(), o.ID); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.AvailableBalance.Equal(decimal.RequireFromString("200")) || !got.LockBalance.Equal(decimal.RequireFromString("1800")) {
		t.Fatalf("after pay avail=%s lock=%s", got.AvailableBalance, got.LockBalance)
	}
	if !got.IspayBalance.Equal(decimal.RequireFromString("0.2")) || !got.LockIspay.Equal(decimal.RequireFromString("1.8")) {
		t.Fatalf("after pay ispay=%s lock_ispay=%s", got.IspayBalance, got.LockIspay)
	}

	uc.now = func() time.Time { return shanghaiNoon("2026-09-12") }
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	got, err = users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("1800")) {
		t.Fatalf("daily settle must skip inactive freeze, lock=%s", got.LockBalance)
	}
	if !got.AvailableBalance.Equal(decimal.RequireFromString("200")) {
		t.Fatalf("daily settle must not credit leftover, avail=%s", got.AvailableBalance)
	}

	uc.now = func() time.Time { return shanghaiStart("2026-09-15") }
	n, err := uc.ExpireCapOverflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("burned=%d", n)
	}
	got, err = users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.IsZero() {
		t.Fatalf("after 72h lock=%s", got.LockBalance)
	}
	if !got.AvailableBalance.Equal(decimal.RequireFromString("200")) {
		t.Fatalf("clear must not credit available, avail=%s", got.AvailableBalance)
	}
}

func TestAdjust_LockFollowsOverflow72h(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:      "0xdddddddddddddddddddddddddddddddddddddddd",
		PaidAmount:   decimal.RequireFromString("1000"),
		CapEffective: decimal.RequireFromString("600"),
	})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, u.ID, "1000", "2026-09-13")
	led := &memLedger{}
	now := shanghaiNoon("2026-09-11")
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, &memConfigs{min: "10"}, true, now)
	adj := NewAdjustUseCase(users, users, led, NopTx{}, uc.daily)
	adj.now = func() time.Time { return now }
	if _, err := adj.Adjust(context.Background(), u.Address, AdjustLock, decimal.RequireFromString("2000")); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("2000")) {
		t.Fatalf("lock=%s", got.LockBalance)
	}

	uc.now = func() time.Time { return shanghaiNoon("2026-09-12") }
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	got, err = users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("2000")) || !got.AvailableBalance.IsZero() {
		t.Fatalf("daily settle must skip admin lock, lock=%s avail=%s", got.LockBalance, got.AvailableBalance)
	}

	uc.now = func() time.Time { return shanghaiStart("2026-09-15") }
	n, err := uc.ExpireCapOverflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("burned=%d", n)
	}
	got, err = users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.IsZero() {
		t.Fatalf("after 72h lock=%s", got.LockBalance)
	}
	if !got.AvailableBalance.IsZero() {
		t.Fatalf("clear must not credit available, avail=%s", got.AvailableBalance)
	}
}

func TestComputeLockUnlock(t *testing.T) {
	inactive := &User{
		LockBalance:  decimal.RequireFromString("1000"),
		LockIspay:    decimal.RequireFromString("1"),
		CapEffective: decimal.RequireFromString("600"),
	}
	p := ComputeLockUnlock(inactive, false)
	if !p.UnlockTodayUSDT.IsZero() || !p.UnlockTodayIspay.IsZero() {
		t.Fatalf("inactive unlock usdt=%s ispay=%s", p.UnlockTodayUSDT, p.UnlockTodayIspay)
	}
	if !p.LockUSDT.Equal(decimal.RequireFromString("1000")) {
		t.Fatalf("lock=%s", p.LockUSDT)
	}

	active := &User{
		PaidAmount:   decimal.RequireFromString("1000"),
		LockBalance:  decimal.RequireFromString("1000"),
		LockIspay:    decimal.RequireFromString("1"),
		CapEffective: decimal.RequireFromString("600"),
	}
	p = ComputeLockUnlock(active, false)
	if !p.UnlockTodayUSDT.Equal(decimal.RequireFromString("600")) || !p.UnlockTodayIspay.Equal(decimal.RequireFromString("0.6")) {
		t.Fatalf("capped unlock usdt=%s ispay=%s", p.UnlockTodayUSDT, p.UnlockTodayIspay)
	}
	p = ComputeLockUnlock(active, true)
	if !p.UnlockTodayUSDT.IsZero() || !p.TodayReleased {
		t.Fatalf("released unlock=%s", p.UnlockTodayUSDT)
	}

	small := &User{
		PaidAmount:   decimal.RequireFromString("1000"),
		LockBalance:  decimal.RequireFromString("100"),
		LockIspay:    decimal.RequireFromString("0.1"),
		CapEffective: decimal.RequireFromString("600"),
	}
	p = ComputeLockUnlock(small, false)
	if !p.UnlockTodayUSDT.Equal(decimal.RequireFromString("100")) || !p.UnlockTodayIspay.Equal(decimal.RequireFromString("0.1")) {
		t.Fatalf("full unlock usdt=%s ispay=%s", p.UnlockTodayUSDT, p.UnlockTodayIspay)
	}
}

func TestPreviewUserLockUnlock_TodayReleased(t *testing.T) {
	u := &User{
		ID:           7,
		PaidAmount:   decimal.RequireFromString("1000"),
		LockBalance:  decimal.RequireFromString("1000"),
		LockIspay:    decimal.RequireFromString("1"),
		CapEffective: decimal.RequireFromString("600"),
	}
	day := time.Date(2026, 9, 13, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	led := &memLedger{}
	if err := led.Create(context.Background(), &LedgerEntry{
		UserID: u.ID, EntryType: LedgerActivate, Amount: decimal.RequireFromString("600"),
		BalanceKind: BalanceLock, SettleDate: &day, Remark: remarkDailyLockUSDT,
	}); err != nil {
		t.Fatal(err)
	}
	uc := &SettleUseCase{
		ledger:   led,
		timezone: "Asia/Shanghai",
		now:      func() time.Time { return day.Add(12 * time.Hour) },
	}
	p, err := uc.PreviewUserLockUnlock(context.Background(), u)
	if err != nil {
		t.Fatal(err)
	}
	if !p.TodayReleased || !p.UnlockTodayUSDT.IsZero() {
		t.Fatalf("today released unlock=%s released=%v", p.UnlockTodayUSDT, p.TodayReleased)
	}
	p, err = uc.PreviewUserLockUnlock(context.Background(), &User{
		ID:           8,
		PaidAmount:   decimal.RequireFromString("1000"),
		LockBalance:  decimal.RequireFromString("1000"),
		CapEffective: decimal.RequireFromString("600"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.TodayReleased || !p.UnlockTodayUSDT.Equal(decimal.RequireFromString("600")) {
		t.Fatalf("other user unlock=%s released=%v", p.UnlockTodayUSDT, p.TodayReleased)
	}
}
