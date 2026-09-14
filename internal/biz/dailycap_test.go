package biz

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestSplitDailyCap(t *testing.T) {
	cases := []struct {
		full, cap, used, under, overflow string
	}{
		{"100", "600", "0", "100", "0"},
		{"100", "600", "550", "50", "50"},
		{"100", "600", "600", "0", "100"},
		{"100", "0", "0", "0", "100"},
		{"100", "600", "700", "0", "100"},
	}
	for _, tc := range cases {
		under, overflow := SplitDailyCap(
			decimal.RequireFromString(tc.full),
			decimal.RequireFromString(tc.cap),
			decimal.RequireFromString(tc.used),
		)
		if !under.Equal(decimal.RequireFromString(tc.under)) || !overflow.Equal(decimal.RequireFromString(tc.overflow)) {
			t.Fatalf("full=%s cap=%s used=%s under=%s overflow=%s", tc.full, tc.cap, tc.used, under, overflow)
		}
	}
}

func TestOverflowClearAt_Hours(t *testing.T) {
	day := shanghaiStart("2026-09-11")
	got := overflowClearAt(day, 72)
	if !got.Equal(shanghaiStart("2026-09-15")) {
		t.Fatalf("72h got %s", got)
	}
	got = overflowClearAt(day, 36)
	want := shanghaiStart("2026-09-13").Add(12 * time.Hour)
	if !got.Equal(want) {
		t.Fatalf("36h got %s want %s", got, want)
	}
}

func TestMatchLedgerCredit(t *testing.T) {
	e := &LedgerEntry{Amount: decimal.RequireFromString("50"), Remark: "match pair=1000 rate=0.1 credit=100 cap=600"}
	if !MatchLedgerCredit(e).Equal(decimal.RequireFromString("100")) {
		t.Fatalf("got %s", MatchLedgerCredit(e))
	}
	e = &LedgerEntry{Amount: decimal.RequireFromString("250"), Remark: "match pair=5000 rate=0.1 cap=600"}
	if !MatchLedgerCredit(e).Equal(decimal.RequireFromString("500")) {
		t.Fatalf("pair*rate got %s", MatchLedgerCredit(e))
	}
}

func TestDailyCap_AccumulatesDirectOverflow(t *testing.T) {
	users := newMemUsers()
	inv, err := users.Create(context.Background(), &User{
		Address:      "0xinvcap",
		PaidAmount:   decimal.RequireFromString("1000"),
		CapEffective: decimal.RequireFromString("600"),
	})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, inv.ID, "1000", "2026-09-10")
	invID := inv.ID
	for i := 0; i < 7; i++ {
		buyer, err := users.Create(context.Background(), &User{
			Address:   "0xbuy" + decimal.NewFromInt(int64(i)).String(),
			InviterID: &invID,
		})
		if err != nil {
			t.Fatal(err)
		}
		mustPay(t, orders, buyer.ID, "1000", "2026-09-11")
	}
	led := &memLedger{}
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, &memConfigs{min: "10"}, true, shanghaiNoon("2026-09-11"))
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(context.Background(), inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("50")) {
		t.Fatalf("overflow lock=%s avail=%s", got.LockBalance, got.AvailableBalance)
	}
	used, err := uc.daily.GetUsed(context.Background(), inv.ID, shanghaiNoon("2026-09-11"))
	if err != nil {
		t.Fatal(err)
	}
	if !used.Equal(decimal.RequireFromString("600")) {
		t.Fatalf("used=%s", used)
	}
}

func TestDailyCap_LargerOrderReleasesOverflow(t *testing.T) {
	users := newMemUsers()
	inv, err := users.Create(context.Background(), &User{
		Address:      "0xinvup",
		PaidAmount:   decimal.RequireFromString("1000"),
		CapEffective: decimal.RequireFromString("600"),
	})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, inv.ID, "1000", "2026-09-10")
	invID := inv.ID
	buyer, err := users.Create(context.Background(), &User{Address: "0xbigbuy", InviterID: &invID})
	if err != nil {
		t.Fatal(err)
	}
	mustPay(t, orders, buyer.ID, "7000", "2026-09-11")
	led := &memLedger{}
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, &memConfigs{min: "10"}, true, shanghaiNoon("2026-09-11"))
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(context.Background(), inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("50")) {
		t.Fatalf("before raise lock=%s", got.LockBalance)
	}
	ouc := NewOrderUseCase(&memPackages{rows: seedCapPackages()}, orders, users, users, led)
	ouc.SetPaidHook(uc)
	o, err := ouc.CreateOrder(context.Background(), inv.ID, decimal.RequireFromString("3000"), 300)
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
	if !got.LockBalance.IsZero() {
		t.Fatalf("after larger order lock=%s", got.LockBalance)
	}
	used, err := uc.daily.GetUsed(context.Background(), inv.ID, shanghaiNoon("2026-09-11"))
	if err != nil {
		t.Fatal(err)
	}
	if !used.Equal(decimal.RequireFromString("600")) {
		t.Fatalf("used after order unlock=%s", used)
	}
}

func TestDailyCap_OverflowClearsAfter72hFromSettle(t *testing.T) {
	users := newMemUsers()
	inv, err := users.Create(context.Background(), &User{Address: "0xburn"})
	if err != nil {
		t.Fatal(err)
	}
	invID := inv.ID
	buyer, err := users.Create(context.Background(), &User{Address: "0xburnb", InviterID: &invID})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, buyer.ID, "1000", "2026-09-11")
	led := &memLedger{}
	now := shanghaiNoon("2026-09-11")
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, &memConfigs{min: "10"}, true, now)
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(context.Background(), inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("50")) {
		t.Fatalf("lock=%s", got.LockBalance)
	}
	avail := got.AvailableBalance
	uc.now = func() time.Time { return shanghaiNoon("2026-09-12") }
	n, err := uc.ExpireCapOverflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("unpackaged/24h must not burn, burned=%d", n)
	}
	uc.now = func() time.Time { return shanghaiNoon("2026-09-14") }
	n, err = uc.ExpireCapOverflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("before 17th-equivalent zero time must not burn, burned=%d", n)
	}
	uc.now = func() time.Time { return shanghaiStart("2026-09-15") }
	n, err = uc.ExpireCapOverflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("burned=%d", n)
	}
	got, err = users.FindByID(context.Background(), inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.IsZero() {
		t.Fatalf("after 72h lock=%s", got.LockBalance)
	}
	if !got.AvailableBalance.Equal(avail) {
		t.Fatalf("clear must not credit available, avail=%s want=%s", got.AvailableBalance, avail)
	}
	if !hasLedgerRemark(led, inv.ID, LedgerActivate, remarkOverflowClear72h) {
		t.Fatal("missing overflow clear 72h unfreeze ledger")
	}
}

func TestDailyCap_OverflowClearHoursFromConfig(t *testing.T) {
	users := newMemUsers()
	inv, err := users.Create(context.Background(), &User{Address: "0xburndays"})
	if err != nil {
		t.Fatal(err)
	}
	invID := inv.ID
	buyer, err := users.Create(context.Background(), &User{Address: "0xburndaysb", InviterID: &invID})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, buyer.ID, "1000", "2026-09-11")
	led := &memLedger{}
	now := shanghaiNoon("2026-09-11")
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, &memConfigs{min: "10", overflowHours: "24"}, true, now)
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	uc.now = func() time.Time { return shanghaiNoon("2026-09-12") }
	n, err := uc.ExpireCapOverflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("day+1 must not burn, burned=%d", n)
	}
	uc.now = func() time.Time { return shanghaiStart("2026-09-13") }
	n, err = uc.ExpireCapOverflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("hours=24 should burn 24h after stamp, burned=%d", n)
	}
	got, err := users.FindByID(context.Background(), inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.IsZero() {
		t.Fatalf("after 24h lock=%s", got.LockBalance)
	}
}

func TestDailyCap_ForceSettleClearsOverflowAfter4Days(t *testing.T) {
	users := newMemUsers()
	inv, err := users.Create(context.Background(), &User{Address: "0xforceburn"})
	if err != nil {
		t.Fatal(err)
	}
	invID := inv.ID
	buyer, err := users.Create(context.Background(), &User{Address: "0xforceburnb", InviterID: &invID})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, buyer.ID, "1000", "2026-09-11")
	led := &memLedger{}
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, &memConfigs{min: "10"}, true, shanghaiNoon("2026-09-11"))
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(context.Background(), inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("50")) {
		t.Fatalf("lock=%s", got.LockBalance)
	}
	avail := got.AvailableBalance
	for _, day := range []string{"2026-09-12", "2026-09-13", "2026-09-14"} {
		res, err := uc.Run(context.Background(), true)
		if err != nil {
			t.Fatal(err)
		}
		if res.SettleDate != day {
			t.Fatalf("force date=%s want=%s", res.SettleDate, day)
		}
		if res.OverflowCleared != 0 {
			t.Fatalf("%s burned=%d", day, res.OverflowCleared)
		}
		got, err = users.FindByID(context.Background(), inv.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !got.LockBalance.Equal(decimal.RequireFromString("50")) {
			t.Fatalf("%s lock=%s", day, got.LockBalance)
		}
	}
	res, err := uc.Run(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if res.SettleDate != "2026-09-15" || res.OverflowCleared != 1 {
		t.Fatalf("force 15th date=%s burned=%d", res.SettleDate, res.OverflowCleared)
	}
	got, err = users.FindByID(context.Background(), inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.IsZero() {
		t.Fatalf("after force 72h lock=%s", got.LockBalance)
	}
	if !got.AvailableBalance.Equal(avail) {
		t.Fatalf("clear must not credit available, avail=%s want=%s", got.AvailableBalance, avail)
	}
	if !hasLedgerRemark(led, inv.ID, LedgerActivate, remarkOverflowClear72h) {
		t.Fatal("missing overflow clear 72h unfreeze ledger")
	}
}

func TestDailyCap_SameTierOrderUnlocksOverflow(t *testing.T) {
	users := newMemUsers()
	inv, err := users.Create(context.Background(), &User{
		Address:      "0xsame",
		PaidAmount:   decimal.RequireFromString("1000"),
		CapEffective: decimal.RequireFromString("600"),
	})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, inv.ID, "1000", "2026-09-10")
	invID := inv.ID
	for i := 0; i < 13; i++ {
		buyer, err := users.Create(context.Background(), &User{
			Address:   "0xsameb" + decimal.NewFromInt(int64(i)).String(),
			InviterID: &invID,
		})
		if err != nil {
			t.Fatal(err)
		}
		mustPay(t, orders, buyer.ID, "1000", "2026-09-11")
	}
	led := &memLedger{}
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, &memConfigs{min: "10"}, true, shanghaiNoon("2026-09-11"))
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(context.Background(), inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("350")) {
		t.Fatalf("overflow lock=%s", got.LockBalance)
	}
	used, err := uc.daily.GetUsed(context.Background(), inv.ID, shanghaiNoon("2026-09-11"))
	if err != nil {
		t.Fatal(err)
	}
	if !used.Equal(decimal.RequireFromString("600")) {
		t.Fatalf("used=%s", used)
	}
	ouc := NewOrderUseCase(&memPackages{rows: seedCapPackages()}, orders, users, users, led)
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
	if !got.LockBalance.Equal(decimal.RequireFromString("50")) {
		t.Fatalf("after same-tier buy lock=%s", got.LockBalance)
	}
	used, err = uc.daily.GetUsed(context.Background(), inv.ID, shanghaiNoon("2026-09-11"))
	if err != nil {
		t.Fatal(err)
	}
	if !used.Equal(decimal.RequireFromString("600")) {
		t.Fatalf("used must stay 600, got %s", used)
	}
	if !hasLedgerRemark(led, inv.ID, LedgerActivate, remarkOverflowOrderUnlock) {
		t.Fatal("missing overflow unlock by order cap ledger")
	}
}

func TestDailyCap_DailyUnlockSkipsOverflow(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:      "0xskipov",
		PaidAmount:   decimal.RequireFromString("1000"),
		CapEffective: decimal.RequireFromString("600"),
	})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, u.ID, "1000", "2026-09-10")
	invID := u.ID
	buyer, err := users.Create(context.Background(), &User{Address: "0xskipb", InviterID: &invID})
	if err != nil {
		t.Fatal(err)
	}
	mustPay(t, orders, buyer.ID, "7000", "2026-09-11")
	led := &memLedger{}
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, &memConfigs{min: "10"}, true, shanghaiNoon("2026-09-11"))
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	overflowLock := got.LockBalance
	if !overflowLock.Equal(decimal.RequireFromString("50")) {
		t.Fatalf("lock=%s", overflowLock)
	}
	uc.now = func() time.Time { return shanghaiNoon("2026-09-12").Add(-11 * time.Hour) }
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	got, err = users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.Equal(overflowLock) {
		t.Fatalf("daily unlock must skip overflow, lock=%s", got.LockBalance)
	}
}

func TestInactive_AllRewardsOverflowNotDailyUsed(t *testing.T) {
	users := newMemUsers()
	inv, err := users.Create(context.Background(), &User{Address: "0xinact"})
	if err != nil {
		t.Fatal(err)
	}
	invID := inv.ID
	orders := newMemOrders(users)
	for i := 0; i < 7; i++ {
		buyer, err := users.Create(context.Background(), &User{
			Address:   "0xinactb" + decimal.NewFromInt(int64(i)).String(),
			InviterID: &invID,
		})
		if err != nil {
			t.Fatal(err)
		}
		mustPay(t, orders, buyer.ID, "1000", "2026-09-11")
	}
	led := &memLedger{}
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, &memConfigs{min: "10"}, true, shanghaiNoon("2026-09-11"))
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(context.Background(), inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("350")) || !got.AvailableBalance.IsZero() {
		t.Fatalf("inactive lock=%s avail=%s", got.LockBalance, got.AvailableBalance)
	}
	used, err := uc.daily.GetUsed(context.Background(), inv.ID, shanghaiNoon("2026-09-11"))
	if err != nil {
		t.Fatal(err)
	}
	if !used.IsZero() {
		t.Fatalf("inactive must not consume daily used, used=%s", used)
	}
	ouc := NewOrderUseCase(&memPackages{rows: seedCapPackages()}, orders, users, users, led)
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
	if !got.LockBalance.Equal(decimal.RequireFromString("50")) {
		t.Fatalf("after first buy lock=%s", got.LockBalance)
	}
	if !got.AvailableBalance.Equal(decimal.RequireFromString("300")) {
		t.Fatalf("after first buy avail=%s", got.AvailableBalance)
	}
	uc.now = func() time.Time { return shanghaiNoon("2026-09-12") }
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	got, err = users.FindByID(context.Background(), inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("50")) {
		t.Fatalf("daily settle must skip leftover inactive freeze, lock=%s", got.LockBalance)
	}
}

func shanghaiStart(day string) time.Time {
	return shanghaiNoon(day).Add(-12 * time.Hour)
}

func hasLedgerRemark(led *memLedger, userID uint64, entryType, remark string) bool {
	if led == nil {
		return false
	}
	for _, e := range led.rows {
		if e != nil && e.UserID == userID && e.EntryType == entryType && e.Remark == remark {
			return true
		}
	}
	return false
}
