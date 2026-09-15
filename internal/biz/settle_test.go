package biz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cigc/app/internal/conf"
	"github.com/shopspring/decimal"
)

func seedCapPackages() []*Package {
	return []*Package{
		{ID: 1, Amount: decimal.RequireFromString("1000"), DailyCap: decimal.RequireFromString("600"), Enabled: true},
		{ID: 2, Amount: decimal.RequireFromString("3000"), DailyCap: decimal.RequireFromString("1800"), Enabled: true},
		{ID: 3, Amount: decimal.RequireFromString("6000"), DailyCap: decimal.RequireFromString("4000"), Enabled: true},
		{ID: 4, Amount: decimal.RequireFromString("160000"), DailyCap: decimal.RequireFromString("100000"), Enabled: true},
	}
}

func TestMatchCap(t *testing.T) {
	cases := []struct {
		paid string
		want string
	}{
		{"0", "0"},
		{"1", "600"},
		{"999", "600"},
		{"2999.99999999", "600"},
		{"3000", "1800"},
		{"3999.99999999", "1800"},
		{"5999.99999999", "1800"},
		{"6000", "4000"},
		{"11999.99999999", "4000"},
		{"12000", "16000"},
		{"23999.99999999", "16000"},
		{"24000", "24000"},
		{"35999.99999999", "24000"},
		{"36000", "30000"},
		{"49999.99999999", "30000"},
		{"50000", "42000"},
		{"69999.99999999", "42000"},
		{"70000", "60000"},
		{"99999.99999999", "60000"},
		{"100000", "100000"},
		{"160000", "100000"},
		{"200000", "100000"},
	}
	for _, tc := range cases {
		got := CapForAmount(decimal.RequireFromString(tc.paid))
		want := decimal.RequireFromString(tc.want)
		if !got.Equal(want) {
			t.Fatalf("paid=%s got=%s want=%s", tc.paid, got, want)
		}
		if !MatchCap(decimal.RequireFromString(tc.paid), nil).Equal(want) {
			t.Fatalf("MatchCap paid=%s", tc.paid)
		}
	}
}

func TestMatchCap_DisabledPackageStillCounts(t *testing.T) {
	got := MatchCap(decimal.RequireFromString("1000"), []*Package{{
		ID: 1, Amount: decimal.RequireFromString("1000"),
		DailyCap: decimal.RequireFromString("1"), Enabled: false,
	}})
	if !got.Equal(decimal.RequireFromString("600")) {
		t.Fatalf("got %s", got)
	}
}

type memSettleRuns struct {
	byDate map[string]*SettleRun
}

func (m *memSettleRuns) key(day time.Time) string {
	return day.Format("2006-01-02")
}

func (m *memSettleRuns) FindByDate(_ context.Context, settleDate time.Time) (*SettleRun, error) {
	if m.byDate == nil {
		return nil, nil
	}
	row, ok := m.byDate[m.key(settleDate)]
	if !ok {
		return nil, nil
	}
	cp := *row
	return &cp, nil
}

func (m *memSettleRuns) FindLatest(_ context.Context) (*SettleRun, error) {
	var latest *SettleRun
	for _, row := range m.byDate {
		if latest == nil || row.SettleDate.After(latest.SettleDate) {
			cp := *row
			latest = &cp
		}
	}
	return latest, nil
}

func (m *memSettleRuns) TryClaim(_ context.Context, settleDate time.Time, remark string) (bool, error) {
	if m.byDate == nil {
		m.byDate = map[string]*SettleRun{}
	}
	k := m.key(settleDate)
	if _, ok := m.byDate[k]; ok {
		return false, nil
	}
	day := time.Date(settleDate.Year(), settleDate.Month(), settleDate.Day(), 0, 0, 0, 0, settleDate.Location())
	m.byDate[k] = &SettleRun{SettleDate: day, Remark: remark}
	return true, nil
}

func (m *memSettleRuns) Upsert(_ context.Context, run *SettleRun) error {
	if m.byDate == nil {
		m.byDate = map[string]*SettleRun{}
	}
	cp := *run
	m.byDate[m.key(run.SettleDate)] = &cp
	return nil
}

func (m *memSettleRuns) DeleteAfter(_ context.Context, after time.Time) (int, error) {
	if m.byDate == nil {
		return 0, nil
	}
	cutoff := m.key(after)
	n := 0
	for k := range m.byDate {
		if k > cutoff {
			delete(m.byDate, k)
			n++
		}
	}
	return n, nil
}

func shanghaiNoon(day string) time.Time {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	t, err := time.ParseInLocation("2006-01-02", day, loc)
	if err != nil {
		panic(err)
	}
	return t.Add(12 * time.Hour)
}

func newSettleUC(users *memUsers, pkgs *memPackages, runs *memSettleRuns, allowForce bool, now time.Time) *SettleUseCase {
	return newSettleUCFull(users, pkgs, runs, newMemOrders(users), &memLedger{}, &memConfigs{min: "10"}, allowForce, now)
}

func newSettleUCFull(users *memUsers, pkgs *memPackages, runs *memSettleRuns, orders *memOrders, led *memLedger, cfg *memConfigs, allowForce bool, now time.Time) *SettleUseCase {
	uc := NewSettleUseCase(users, pkgs, runs, orders, users, led, cfg, nil, nil, nil, NopTx{}, &conf.App{
		SettleTimezone:   "Asia/Shanghai",
		AllowForceSettle: allowForce,
	})
	uc.now = func() time.Time { return now }
	return uc
}

func TestSettle_ForceDisabled(t *testing.T) {
	uc := newSettleUC(newMemUsers(), &memPackages{}, &memSettleRuns{}, false, shanghaiNoon("2026-09-11"))
	_, err := uc.Run(context.Background(), true)
	if !errors.Is(err, ErrForceSettleDisabled) {
		t.Fatalf("got %v", err)
	}
}

func TestSettle_WritesCapFromPaidAmount(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address: "0xabc",
	})
	if err != nil {
		t.Fatal(err)
	}
	disabledAt := shanghaiNoon("2026-09-01")
	locked, err := users.Create(context.Background(), &User{
		Address:    "0xdef",
		DisabledAt: &disabledAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, u.ID, "4000", "2026-09-10")
	mustPay(t, orders, locked.ID, "1000", "2026-09-10")
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, &memLedger{}, &memConfigs{min: "10"}, true, shanghaiNoon("2026-09-11"))
	res, err := uc.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped || res.SettleDate != "2026-09-11" || res.UserCount != 2 || res.CapUpdated != 2 {
		t.Fatalf("%+v", res)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.CapEffective.Equal(decimal.RequireFromString("1800")) {
		t.Fatalf("cap=%s", got.CapEffective)
	}
	got, err = users.FindByID(context.Background(), locked.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.CapEffective.Equal(decimal.RequireFromString("600")) {
		t.Fatalf("locked cap=%s", got.CapEffective)
	}
}

func TestSettle_SameDaySkipped(t *testing.T) {
	users := newMemUsers()
	if _, err := users.Create(context.Background(), &User{
		Address:    "0xabc",
		PaidAmount: decimal.RequireFromString("1000"),
	}); err != nil {
		t.Fatal(err)
	}
	runs := &memSettleRuns{}
	uc := newSettleUC(users, &memPackages{rows: seedCapPackages()}, runs, true, shanghaiNoon("2026-09-11"))
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	res, err := uc.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Skipped || res.SettleDate != "2026-09-11" {
		t.Fatalf("%+v", res)
	}
}

func TestSettle_NewPurchaseRaisesCapImmediately(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	led := &memLedger{}
	pkgs := &memPackages{rows: seedCapPackages()}
	uc := newSettleUCFull(users, pkgs, &memSettleRuns{}, orders, led, &memConfigs{min: "10"}, true, shanghaiNoon("2026-09-11"))
	ouc := NewOrderUseCase(pkgs, orders, users, users, led)
	ouc.SetPaidHook(uc)

	o1, err := ouc.CreateOrder(context.Background(), u.ID, decimal.RequireFromString("1000"), 300)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ouc.MarkPaid(context.Background(), o1.ID); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.CapEffective.Equal(decimal.RequireFromString("600")) {
		t.Fatalf("first pay cap=%s", got.CapEffective)
	}

	o2, err := ouc.CreateOrder(context.Background(), u.ID, decimal.RequireFromString("3000"), 300)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ouc.MarkPaid(context.Background(), o2.ID); err != nil {
		t.Fatal(err)
	}
	got, err = users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.CapEffective.Equal(decimal.RequireFromString("1800")) {
		t.Fatalf("larger pay cap=%s", got.CapEffective)
	}
}

func TestSettle_ForceRerun(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address: "0xabc",
	})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, u.ID, "1000", "2026-09-10")
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, &memLedger{}, &memConfigs{min: "10"}, true, shanghaiNoon("2026-09-11"))
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	mustPay(t, orders, u.ID, "3000", "2026-09-11")
	res, err := uc.Run(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped || !res.Forced {
		t.Fatalf("%+v", res)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.CapEffective.Equal(decimal.RequireFromString("1800")) {
		t.Fatalf("force cap=%s", got.CapEffective)
	}
}

func TestSettle_ResetTestDay(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xreset"})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, u.ID, "1000", "2026-09-10")
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, &memLedger{}, &memConfigs{min: "10"}, true, shanghaiNoon("2026-09-11"))
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Run(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Run(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	st, err := uc.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.NextTestDate != "2026-09-14" {
		t.Fatalf("next=%s", st.NextTestDate)
	}
	reset, err := uc.ResetTestDay(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if reset.TodayDate != "2026-09-11" || reset.Deleted != 2 || reset.NextTestDate != "2026-09-12" {
		t.Fatalf("%+v", reset)
	}
	st, err = uc.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.NextTestDate != "2026-09-12" {
		t.Fatalf("after reset next=%s", st.NextTestDate)
	}
	if !st.TodaySettled {
		t.Fatal("today settle must remain")
	}
}

func TestSettle_ResetTestDayDisabled(t *testing.T) {
	uc := newSettleUC(newMemUsers(), &memPackages{}, &memSettleRuns{}, false, shanghaiNoon("2026-09-11"))
	_, err := uc.ResetTestDay(context.Background())
	if !errors.Is(err, ErrForceSettleDisabled) {
		t.Fatalf("err=%v", err)
	}
}

type memTestData struct {
	users  *memUsers
	orders *memOrders
	led    *memLedger
	runs   *memSettleRuns
	daily  *memDailyCap
	match  *memMatch
}

func (m *memTestData) ClearKeepUsers(_ context.Context) (*TestDataClearResult, error) {
	out := &TestDataClearResult{}
	if m.users != nil {
		out.UsersKept = len(m.users.byID)
		for _, u := range m.users.byID {
			u.AvailableBalance = decimal.Zero
			u.RechargeBalance = decimal.Zero
			u.FrozenBalance = decimal.Zero
			u.FrozenIspay = decimal.Zero
			u.IspayBalance = decimal.Zero
			u.LockBalance = decimal.Zero
			u.LockIspay = decimal.Zero
			u.PaidAmount = decimal.Zero
			u.CapEffective = decimal.Zero
			u.DisabledAt = nil
		}
	}
	if m.orders != nil {
		out.OrdersCleared = len(m.orders.byID)
		m.orders.byID = map[uint64]*Order{}
	}
	if m.led != nil {
		out.LedgerCleared = len(m.led.rows)
		m.led.rows = nil
	}
	if m.runs != nil {
		m.runs.byDate = map[string]*SettleRun{}
	}
	if m.daily != nil {
		out.HoldsCleared = len(m.daily.holds)
		m.daily.used = map[string]decimal.Decimal{}
		m.daily.holds = nil
	}
	if m.match != nil {
		m.match.bals = map[uint64]*MatchBalance{}
		m.match.applied = map[uint64]struct{}{}
	}
	return out, nil
}

func TestSettle_ClearTestDataDisabled(t *testing.T) {
	uc := newSettleUC(newMemUsers(), &memPackages{}, &memSettleRuns{}, false, shanghaiNoon("2026-09-11"))
	uc.SetTestData(&memTestData{})
	_, err := uc.ClearTestData(context.Background())
	if !errors.Is(err, ErrForceSettleDisabled) {
		t.Fatalf("err=%v", err)
	}
}

func TestSettle_ClearTestDataKeepsUsers(t *testing.T) {
	users := newMemUsers()
	inv := uint64(1)
	u1, err := users.Create(context.Background(), &User{Address: "0xkeep1"})
	if err != nil {
		t.Fatal(err)
	}
	u2, err := users.Create(context.Background(), &User{
		Address:          "0xkeep2",
		InviterID:        &inv,
		AvailableBalance: decimal.RequireFromString("100"),
		LockBalance:      decimal.RequireFromString("50"),
		PaidAmount:       decimal.RequireFromString("1000"),
		CapEffective:     decimal.RequireFromString("600"),
	})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, u1.ID, "1000", "2026-09-10")
	led := &memLedger{}
	if err := led.Create(context.Background(), &LedgerEntry{UserID: u1.ID, EntryType: "static", Amount: decimal.RequireFromString("1")}); err != nil {
		t.Fatal(err)
	}
	runs := &memSettleRuns{}
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, runs, orders, led, &memConfigs{min: "10"}, true, shanghaiNoon("2026-09-11"))
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	daily, ok := uc.daily.(*memDailyCap)
	if !ok {
		t.Fatal("daily cap is not mem")
	}
	uc.SetTestData(&memTestData{users: users, orders: orders, led: led, runs: runs, daily: daily})
	res, err := uc.ClearTestData(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.UsersKept != 2 {
		t.Fatalf("users=%d", res.UsersKept)
	}
	if res.OrdersCleared < 1 {
		t.Fatalf("orders=%d", res.OrdersCleared)
	}
	got1, err := users.FindByID(context.Background(), u1.ID)
	if err != nil {
		t.Fatal(err)
	}
	got2, err := users.FindByID(context.Background(), u2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got1.Address != "0xkeep1" || got2.Address != "0xkeep2" {
		t.Fatalf("addresses lost")
	}
	if got2.InviterID == nil || *got2.InviterID != 1 {
		t.Fatalf("inviter=%v", got2.InviterID)
	}
	if !got2.AvailableBalance.IsZero() || !got2.LockBalance.IsZero() || !got2.PaidAmount.IsZero() || !got2.CapEffective.IsZero() {
		t.Fatalf("balances not cleared: %+v", got2)
	}
	if len(orders.byID) != 0 {
		t.Fatalf("orders left=%d", len(orders.byID))
	}
	if len(led.rows) != 0 {
		t.Fatalf("ledger left=%d", len(led.rows))
	}
	latest, err := runs.FindLatest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if latest != nil {
		t.Fatalf("settle run left=%+v", latest)
	}
}

func TestSettle_ForceCalendarNewFreezeExpiresLater(t *testing.T) {
	ctx := context.Background()
	users := newMemUsers()
	u, err := users.Create(ctx, &User{
		Address:      "0xcal",
		PaidAmount:   decimal.RequireFromString("1000"),
		CapEffective: decimal.RequireFromString("600"),
	})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, u.ID, "1000", "2026-09-10")
	led := &memLedger{}
	runs := &memSettleRuns{}
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, runs, orders, led, &memConfigs{min: "10"}, true, shanghaiNoon("2026-09-15"))
	if err := uc.creditOverflow(ctx, u.ID, nil, LedgerDirect, LedgerDirectIspay, decimal.RequireFromString("100"), nil, "d15"); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Run(ctx, true); err != nil {
		t.Fatal(err)
	}
	if got := uc.currentBusinessDay(ctx).Format("2006-01-02"); got != "2026-09-16" {
		t.Fatalf("business=%s", got)
	}
	if err := uc.creditOverflow(ctx, u.ID, nil, LedgerDirect, LedgerDirectIspay, decimal.RequireFromString("80"), nil, "d16"); err != nil {
		t.Fatal(err)
	}
	holds, err := uc.daily.ListActiveHolds(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(holds) != 2 {
		t.Fatalf("holds=%d", len(holds))
	}
	byRemark := map[string]string{}
	for _, h := range holds {
		byRemark[h.Remark] = dailyCapDate(h.SettleDate).Format("2006-01-02")
	}
	if byRemark["d15"] != "2026-09-15" || byRemark["d16"] != "2026-09-16" {
		t.Fatalf("%v", byRemark)
	}
	for i := 0; i < 3; i++ {
		if _, err := uc.Run(ctx, true); err != nil {
			t.Fatal(err)
		}
	}
	holds, err = uc.daily.ListActiveHolds(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(holds) != 1 || holds[0].Remark != "d16" {
		t.Fatalf("after 19 active=%+v", holds)
	}
	if _, err := uc.Run(ctx, true); err != nil {
		t.Fatal(err)
	}
	holds, err = uc.daily.ListActiveHolds(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(holds) != 0 {
		t.Fatalf("after 20 still %+v", holds)
	}
}

func TestSettle_CapUsesMaxOrderNotSum(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xmax"})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, u.ID, "1000", "2026-09-09")
	mustPay(t, orders, u.ID, "1000", "2026-09-10")
	mustPay(t, orders, u.ID, "1000", "2026-09-11")
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, &memLedger{}, &memConfigs{min: "10"}, true, shanghaiNoon("2026-09-11"))
	if _, err := uc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.PaidAmount.Equal(decimal.RequireFromString("3000")) {
		t.Fatalf("paid_amount=%s", got.PaidAmount)
	}
	if !got.CapEffective.Equal(decimal.RequireFromString("600")) {
		t.Fatalf("max-order cap=%s", got.CapEffective)
	}
}

func TestSettle_PaysDirectReward(t *testing.T) {
	users := newMemUsers()
	inv, err := users.Create(context.Background(), &User{Address: "0xinv"})
	if err != nil {
		t.Fatal(err)
	}
	invID := inv.ID
	buyer, err := users.Create(context.Background(), &User{Address: "0xbuy", InviterID: &invID})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	paidAt := shanghaiNoon("2026-09-11")
	o, err := orders.Create(context.Background(), &Order{
		UserID: buyer.ID, PackageID: 1, Amount: decimal.RequireFromString("3000"),
		TitleSnapshot: "t", Status: OrderPending,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := orders.MarkPaid(context.Background(), o.ID, buyer.ID, o.Amount, paidAt); err != nil {
		t.Fatal(err)
	}
	// 锁定推荐人仍应发奖
	now := time.Now()
	if err := users.SetDisabledAt(context.Background(), inv.ID, &now); err != nil {
		t.Fatal(err)
	}

	led := &memLedger{}
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, &memConfigs{}, true, paidAt)
	res, err := uc.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.DirectCount != 1 {
		t.Fatalf("direct_count=%d", res.DirectCount)
	}
	got, err := users.FindByID(context.Background(), inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.AvailableBalance.IsZero() || !got.IspayBalance.IsZero() {
		t.Fatalf("inactive should not credit available avail=%s ispay=%s", got.AvailableBalance, got.IspayBalance)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("150")) {
		t.Fatalf("inviter lock=%s", got.LockBalance)
	}
	if !got.LockIspay.Equal(decimal.RequireFromString("0.075")) {
		t.Fatalf("inviter lock ispay=%s", got.LockIspay)
	}
	if !ledgerHasType(led, LedgerDirect) || !ledgerHasType(led, LedgerDirectIspay) {
		t.Fatalf("ledger=%+v", led.rows)
	}

	// force 同日再跑不双发
	res2, err := uc.Run(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if res2.DirectCount != 0 {
		t.Fatalf("force direct_count=%d", res2.DirectCount)
	}
	got, err = users.FindByID(context.Background(), inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("150")) {
		t.Fatalf("after force lock=%s", got.LockBalance)
	}
	if !got.LockIspay.Equal(decimal.RequireFromString("0.075")) {
		t.Fatalf("after force lock ispay=%s", got.LockIspay)
	}
}

func ledgerHasType(led *memLedger, typ string) bool {
	for _, e := range led.rows {
		if e.EntryType == typ {
			return true
		}
	}
	return false
}

type memMatch struct {
	bals    map[uint64]*MatchBalance
	applied map[uint64]struct{}
}

func newMemMatch() *memMatch {
	return &memMatch{bals: map[uint64]*MatchBalance{}, applied: map[uint64]struct{}{}}
}

func (m *memMatch) Get(_ context.Context, userID uint64) (*MatchBalance, error) {
	if b, ok := m.bals[userID]; ok {
		cp := *b
		return &cp, nil
	}
	return &MatchBalance{UserID: userID}, nil
}

func (m *memMatch) AddRemain(_ context.Context, userID uint64, side string, delta decimal.Decimal) error {
	b := m.bals[userID]
	if b == nil {
		b = &MatchBalance{UserID: userID}
		m.bals[userID] = b
	}
	switch side {
	case SideLeft:
		b.LeftRemain = b.LeftRemain.Add(delta)
	case SideRight:
		b.RightRemain = b.RightRemain.Add(delta)
	default:
		return ErrPlacementInvalidSide
	}
	return nil
}

func (m *memMatch) SaveRemains(_ context.Context, userID uint64, left, right decimal.Decimal) error {
	b := m.bals[userID]
	if b == nil {
		b = &MatchBalance{UserID: userID}
		m.bals[userID] = b
	}
	b.LeftRemain = left
	b.RightRemain = right
	return nil
}

func (m *memMatch) ListAll(_ context.Context) ([]*MatchBalance, error) {
	out := make([]*MatchBalance, 0, len(m.bals))
	for _, b := range m.bals {
		cp := *b
		out = append(out, &cp)
	}
	return out, nil
}

func (m *memMatch) TryApplyOrder(_ context.Context, orderID uint64, _ time.Time) (bool, error) {
	if _, ok := m.applied[orderID]; ok {
		return false, nil
	}
	m.applied[orderID] = struct{}{}
	return true, nil
}

func newSettleUCMatch(users *memUsers, pkgs *memPackages, runs *memSettleRuns, orders *memOrders, led *memLedger, place *memPlacements, matches *memMatch, allowForce bool, now time.Time) *SettleUseCase {
	return newSettleUCMatchCfg(users, pkgs, runs, orders, led, place, matches, &memConfigs{}, allowForce, now)
}

func newSettleUCMatchCfg(users *memUsers, pkgs *memPackages, runs *memSettleRuns, orders *memOrders, led *memLedger, place *memPlacements, matches *memMatch, cfg *memConfigs, allowForce bool, now time.Time) *SettleUseCase {
	if cfg == nil {
		cfg = &memConfigs{}
	}
	uc := NewSettleUseCase(users, pkgs, runs, orders, users, led, cfg, place, matches, nil, NopTx{}, &conf.App{
		SettleTimezone:   "Asia/Shanghai",
		AllowForceSettle: allowForce,
	})
	uc.now = func() time.Time { return now }
	return uc
}

func mustPay(t *testing.T, orders *memOrders, userID uint64, amount, day string) {
	t.Helper()
	paidAt := shanghaiNoon(day)
	o, err := orders.Create(context.Background(), &Order{
		UserID: userID, PackageID: 1, Amount: decimal.RequireFromString(amount), Status: OrderPending,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := orders.MarkPaid(context.Background(), o.ID, userID, o.Amount, paidAt); err != nil {
		t.Fatal(err)
	}
}

func TestMatchPair(t *testing.T) {
	credit, pair, l, r := MatchPair(
		decimal.RequireFromString("1000"),
		decimal.RequireFromString("400"),
		decimal.RequireFromString("0.10"),
	)
	if !pair.Equal(decimal.RequireFromString("400")) || !credit.Equal(decimal.RequireFromString("40")) {
		t.Fatalf("credit=%s pair=%s", credit, pair)
	}
	if !l.Equal(decimal.RequireFromString("600")) || !r.Equal(decimal.Zero) {
		t.Fatalf("remain L=%s R=%s", l, r)
	}
	credit, pair, l, r = MatchPair(
		decimal.RequireFromString("1000"),
		decimal.RequireFromString("1000"),
		decimal.RequireFromString("0.10"),
	)
	if !pair.Equal(decimal.RequireFromString("1000")) || !credit.Equal(decimal.RequireFromString("100")) {
		t.Fatalf("credit=%s pair=%s", credit, pair)
	}
	if !l.IsZero() || !r.IsZero() {
		t.Fatalf("pair consume L=%s R=%s", l, r)
	}
}

func TestSettle_PaysMatchFromPlacementTree(t *testing.T) {
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{
		Address: "0xa", CapEffective: decimal.RequireFromString("600"),
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := users.Create(context.Background(), &User{Address: "0xb"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := users.Create(context.Background(), &User{Address: "0xc"})
	if err != nil {
		t.Fatal(err)
	}
	place := newMemPlacements()
	puc := NewPlacementUseCase(users, place, nil)
	if _, err := puc.Place(context.Background(), b.ID, a.ID, SideLeft); err != nil {
		t.Fatal(err)
	}
	if _, err := puc.Place(context.Background(), c.ID, a.ID, SideRight); err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, b.ID, "400", "2026-09-11")
	mustPay(t, orders, c.ID, "1000", "2026-09-11")

	led := &memLedger{}
	matches := newMemMatch()
	uc := newSettleUCMatch(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, place, matches, true, shanghaiNoon("2026-09-11"))
	res, err := uc.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.MatchCount != 1 {
		t.Fatalf("match_count=%d", res.MatchCount)
	}
	got, err := users.FindByID(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.AvailableBalance.IsZero() {
		t.Fatalf("inactive A avail=%s", got.AvailableBalance)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("20")) {
		t.Fatalf("A lock=%s", got.LockBalance)
	}
	bal, err := matches.Get(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !bal.LeftRemain.IsZero() || !bal.RightRemain.Equal(decimal.RequireFromString("600")) {
		t.Fatalf("remain L=%s R=%s", bal.LeftRemain, bal.RightRemain)
	}

	res2, err := uc.Run(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if res2.MatchCount != 0 {
		t.Fatalf("force match_count=%d", res2.MatchCount)
	}
	got, err = users.FindByID(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("20")) {
		t.Fatalf("after force lock=%s", got.LockBalance)
	}
	bal, err = matches.Get(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !bal.RightRemain.Equal(decimal.RequireFromString("600")) {
		t.Fatalf("force remain R=%s", bal.RightRemain)
	}
}

func TestSettle_MatchNestedAndCarryNextDay(t *testing.T) {
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{
		Address:      "0xa",
		PaidAmount:   decimal.RequireFromString("1000"),
		CapEffective: decimal.RequireFromString("600"),
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := users.Create(context.Background(), &User{
		Address: "0xb", CapEffective: decimal.RequireFromString("600"),
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err := users.Create(context.Background(), &User{Address: "0xc"})
	if err != nil {
		t.Fatal(err)
	}
	d, err := users.Create(context.Background(), &User{Address: "0xd"})
	if err != nil {
		t.Fatal(err)
	}
	place := newMemPlacements()
	puc := NewPlacementUseCase(users, place, nil)
	if _, err := puc.Place(context.Background(), b.ID, a.ID, SideLeft); err != nil {
		t.Fatal(err)
	}
	if _, err := puc.Place(context.Background(), c.ID, a.ID, SideRight); err != nil {
		t.Fatal(err)
	}
	if _, err := puc.Place(context.Background(), d.ID, b.ID, SideLeft); err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, a.ID, "1000", "2026-09-10")
	mustPay(t, orders, d.ID, "1000", "2026-09-11")

	led := &memLedger{}
	matches := newMemMatch()
	runs := &memSettleRuns{}
	uc := newSettleUCMatch(users, &memPackages{rows: seedCapPackages()}, runs, orders, led, place, matches, true, shanghaiNoon("2026-09-11"))
	res, err := uc.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.MatchCount != 0 {
		t.Fatalf("day1 match=%d", res.MatchCount)
	}
	aBal, _ := matches.Get(context.Background(), a.ID)
	bBal, _ := matches.Get(context.Background(), b.ID)
	if !aBal.LeftRemain.Equal(decimal.RequireFromString("1000")) || !aBal.RightRemain.IsZero() {
		t.Fatalf("A day1 L=%s R=%s", aBal.LeftRemain, aBal.RightRemain)
	}
	if !bBal.LeftRemain.Equal(decimal.RequireFromString("1000")) {
		t.Fatalf("B day1 L=%s", bBal.LeftRemain)
	}

	mustPay(t, orders, c.ID, "400", "2026-09-12")
	uc.now = func() time.Time { return shanghaiNoon("2026-09-12") }
	res, err = uc.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.MatchCount != 1 {
		t.Fatalf("day2 match=%d", res.MatchCount)
	}
	got, err := users.FindByID(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.AvailableBalance.Equal(decimal.RequireFromString("20")) {
		t.Fatalf("A day2 bal=%s", got.AvailableBalance)
	}
	aBal, _ = matches.Get(context.Background(), a.ID)
	if !aBal.LeftRemain.Equal(decimal.RequireFromString("600")) || !aBal.RightRemain.IsZero() {
		t.Fatalf("A day2 L=%s R=%s", aBal.LeftRemain, aBal.RightRemain)
	}
}

func TestSettle_ZeroCapOverflowKeepsPairBurn(t *testing.T) {
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{
		Address: "0xa", CapEffective: decimal.Zero,
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := users.Create(context.Background(), &User{Address: "0xb"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := users.Create(context.Background(), &User{Address: "0xc"})
	if err != nil {
		t.Fatal(err)
	}
	place := newMemPlacements()
	puc := NewPlacementUseCase(users, place, nil)
	if _, err := puc.Place(context.Background(), b.ID, a.ID, SideLeft); err != nil {
		t.Fatal(err)
	}
	if _, err := puc.Place(context.Background(), c.ID, a.ID, SideRight); err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, b.ID, "1000", "2026-09-11")
	mustPay(t, orders, c.ID, "1000", "2026-09-11")
	led := &memLedger{}
	matches := newMemMatch()
	uc := newSettleUCMatch(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, place, matches, true, shanghaiNoon("2026-09-11"))
	res, err := uc.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.MatchCount != 1 {
		t.Fatalf("overflow still counts match, got %d", res.MatchCount)
	}
	got, err := users.FindByID(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.AvailableBalance.IsZero() {
		t.Fatalf("bal=%s", got.AvailableBalance)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("50")) {
		t.Fatalf("overflow lock=%s", got.LockBalance)
	}
	bal, _ := matches.Get(context.Background(), a.ID)
	if !bal.LeftRemain.IsZero() || !bal.RightRemain.IsZero() {
		t.Fatalf("pair still burned L=%s R=%s", bal.LeftRemain, bal.RightRemain)
	}
}

func TestSettle_MatchSkipsOwnPurchase(t *testing.T) {
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{
		Address: "0xa", CapEffective: decimal.RequireFromString("600"),
	})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, a.ID, "1000", "2026-09-11")
	led := &memLedger{}
	matches := newMemMatch()
	uc := newSettleUCMatch(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, newMemPlacements(), matches, true, shanghaiNoon("2026-09-11"))
	res, err := uc.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.MatchCount != 0 {
		t.Fatalf("match=%d", res.MatchCount)
	}
}

func TestSettle_NoDirectWithoutInviter(t *testing.T) {
	users := newMemUsers()
	buyer, err := users.Create(context.Background(), &User{Address: "0xgenesis"})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	paidAt := shanghaiNoon("2026-09-11")
	o, err := orders.Create(context.Background(), &Order{
		UserID: buyer.ID, PackageID: 1, Amount: decimal.RequireFromString("1000"),
		Status: OrderPending,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := orders.MarkPaid(context.Background(), o.ID, buyer.ID, o.Amount, paidAt); err != nil {
		t.Fatal(err)
	}
	led := &memLedger{}
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, &memConfigs{}, true, paidAt)
	res, err := uc.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.DirectCount != 0 || len(led.rows) != 0 {
		t.Fatalf("direct=%d ledger=%d", res.DirectCount, len(led.rows))
	}
}

func TestManageShares(t *testing.T) {
	s := ManageShares(decimal.RequireFromString("12"), 3)
	if len(s) != 3 || !s[0].Equal(decimal.RequireFromString("4")) || !s[1].Equal(decimal.RequireFromString("4")) || !s[2].Equal(decimal.RequireFromString("4")) {
		t.Fatalf("%v", s)
	}
	s = ManageShares(decimal.RequireFromString("10"), 3)
	if !s[0].Add(s[1]).Add(s[2]).Equal(decimal.RequireFromString("10")) {
		t.Fatalf("sum=%s", s[0].Add(s[1]).Add(s[2]))
	}
	if s[0].LessThan(s[1]) {
		t.Fatalf("remainder should go to gen1: %v", s)
	}
	one := ManageShares(decimal.RequireFromString("10"), 1)
	if len(one) != 1 || !one[0].Equal(decimal.RequireFromString("10")) {
		t.Fatalf("n=1 %v", one)
	}
	two := ManageShares(decimal.RequireFromString("10"), 2)
	if len(two) != 2 || !two[0].Add(two[1]).Equal(decimal.RequireFromString("10")) {
		t.Fatalf("n=2 %v", two)
	}
}

func setupMatchTree(t *testing.T, users *memUsers, matcher *User) (*memPlacements, *memOrders) {
	t.Helper()
	b, err := users.Create(context.Background(), &User{Address: matcher.Address + "l"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := users.Create(context.Background(), &User{Address: matcher.Address + "r"})
	if err != nil {
		t.Fatal(err)
	}
	place := newMemPlacements()
	puc := NewPlacementUseCase(users, place, nil)
	if _, err := puc.Place(context.Background(), b.ID, matcher.ID, SideLeft); err != nil {
		t.Fatal(err)
	}
	if _, err := puc.Place(context.Background(), c.ID, matcher.ID, SideRight); err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	mustPay(t, orders, b.ID, "400", "2026-09-11")
	mustPay(t, orders, c.ID, "1000", "2026-09-11")
	return place, orders
}

func TestSettle_PaysManageFromMatch(t *testing.T) {
	users := newMemUsers()
	i3, err := users.Create(context.Background(), &User{Address: "0xi3"})
	if err != nil {
		t.Fatal(err)
	}
	i2, err := users.Create(context.Background(), &User{Address: "0xi2", InviterID: &i3.ID})
	if err != nil {
		t.Fatal(err)
	}
	i1, err := users.Create(context.Background(), &User{Address: "0xi1", InviterID: &i2.ID})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := users.SetDisabledAt(context.Background(), i1.ID, &now); err != nil {
		t.Fatal(err)
	}
	u, err := users.Create(context.Background(), &User{
		Address: "0xu", InviterID: &i1.ID, CapEffective: decimal.RequireFromString("600"),
	})
	if err != nil {
		t.Fatal(err)
	}
	place, orders := setupMatchTree(t, users, u)
	mustPay(t, orders, i1.ID, "1000", "2026-09-10")
	mustPay(t, orders, i2.ID, "1000", "2026-09-10")
	mustPay(t, orders, i3.ID, "1000", "2026-09-10")
	led := &memLedger{}
	uc := newSettleUCMatch(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, place, newMemMatch(), true, shanghaiNoon("2026-09-11"))
	res, err := uc.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.MatchCount != 1 || res.ManageCount != 3 {
		t.Fatalf("match=%d manage=%d", res.MatchCount, res.ManageCount)
	}
	for _, id := range []uint64{i1.ID, i2.ID, i3.ID} {
		got, err := users.FindByID(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if !got.AvailableBalance.Equal(decimal.RequireFromString("2")) {
			t.Fatalf("user %d avail=%s", id, got.AvailableBalance)
		}
	}

	res2, err := uc.Run(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if res2.ManageCount != 0 {
		t.Fatalf("force manage=%d", res2.ManageCount)
	}
	got, err := users.FindByID(context.Background(), i1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.AvailableBalance.Equal(decimal.RequireFromString("2")) {
		t.Fatalf("after force avail=%s", got.AvailableBalance)
	}
}

func TestSettle_ManageGensFromConfig(t *testing.T) {
	users := newMemUsers()
	i3, err := users.Create(context.Background(), &User{Address: "0xi3g"})
	if err != nil {
		t.Fatal(err)
	}
	i2, err := users.Create(context.Background(), &User{Address: "0xi2g", InviterID: &i3.ID})
	if err != nil {
		t.Fatal(err)
	}
	i1, err := users.Create(context.Background(), &User{Address: "0xi1g", InviterID: &i2.ID})
	if err != nil {
		t.Fatal(err)
	}
	u, err := users.Create(context.Background(), &User{
		Address: "0xug", InviterID: &i1.ID, CapEffective: decimal.RequireFromString("600"),
	})
	if err != nil {
		t.Fatal(err)
	}
	place, orders := setupMatchTree(t, users, u)
	mustPay(t, orders, i1.ID, "1000", "2026-09-10")
	mustPay(t, orders, i2.ID, "1000", "2026-09-10")
	mustPay(t, orders, i3.ID, "1000", "2026-09-10")
	led := &memLedger{}
	uc := newSettleUCMatchCfg(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, place, newMemMatch(), &memConfigs{manageGens: "2"}, true, shanghaiNoon("2026-09-11"))
	res, err := uc.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.MatchCount != 1 || res.ManageCount != 2 {
		t.Fatalf("match=%d manage=%d", res.MatchCount, res.ManageCount)
	}
	g1, _ := users.FindByID(context.Background(), i1.ID)
	g2, _ := users.FindByID(context.Background(), i2.ID)
	g3, _ := users.FindByID(context.Background(), i3.ID)
	if !g1.AvailableBalance.Equal(decimal.RequireFromString("3")) || !g2.AvailableBalance.Equal(decimal.RequireFromString("3")) {
		t.Fatalf("gen1=%s gen2=%s", g1.AvailableBalance, g2.AvailableBalance)
	}
	if !g3.AvailableBalance.IsZero() {
		t.Fatalf("gen3 should be skipped, avail=%s", g3.AvailableBalance)
	}
}

func TestSettle_ManageSkipsMissingGen(t *testing.T) {
	users := newMemUsers()
	i1, err := users.Create(context.Background(), &User{Address: "0xi1"})
	if err != nil {
		t.Fatal(err)
	}
	u, err := users.Create(context.Background(), &User{
		Address: "0xu", InviterID: &i1.ID, CapEffective: decimal.RequireFromString("600"),
	})
	if err != nil {
		t.Fatal(err)
	}
	place, orders := setupMatchTree(t, users, u)
	mustPay(t, orders, i1.ID, "1000", "2026-09-10")
	led := &memLedger{}
	uc := newSettleUCMatch(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, place, newMemMatch(), true, shanghaiNoon("2026-09-11"))
	res, err := uc.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.ManageCount != 1 {
		t.Fatalf("manage=%d", res.ManageCount)
	}
	got, err := users.FindByID(context.Background(), i1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.AvailableBalance.Equal(decimal.RequireFromString("2")) {
		t.Fatalf("should not absorb missing gens, avail=%s", got.AvailableBalance)
	}
}

func TestSettle_PaysStaticRelease(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xstat"})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	paidAt := shanghaiNoon("2026-09-11")
	o, err := orders.Create(context.Background(), &Order{
		UserID: u.ID, PackageID: 1, Amount: decimal.RequireFromString("12000"),
		Status: OrderPending, ReleaseDays: 300,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := orders.MarkPaid(context.Background(), o.ID, u.ID, o.Amount, paidAt); err != nil {
		t.Fatal(err)
	}
	led := &memLedger{}
	uc := newSettleUCFull(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, &memConfigs{}, true, paidAt)
	res, err := uc.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.StaticCount != 1 {
		t.Fatalf("static=%d", res.StaticCount)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.AvailableBalance.Equal(decimal.RequireFromString("33.33333000")) {
		t.Fatalf("usdt=%s", got.AvailableBalance)
	}
	if !got.IspayBalance.Equal(decimal.RequireFromString("0.01666667")) {
		t.Fatalf("ispay=%s", got.IspayBalance)
	}
	res2, err := uc.Run(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if res2.SettleDate != "2026-09-12" {
		t.Fatalf("force date=%s", res2.SettleDate)
	}
	if res2.StaticCount != 1 {
		t.Fatalf("next-day static=%d", res2.StaticCount)
	}
	got, err = users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.AvailableBalance.Equal(decimal.RequireFromString("66.66666000")) {
		t.Fatalf("after next-day usdt=%s", got.AvailableBalance)
	}
}

func TestFindActivatedRecommendAncestors_SkipsInactive(t *testing.T) {
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{Address: "0xa", PaidAmount: decimal.RequireFromString("1000")})
	if err != nil {
		t.Fatal(err)
	}
	b, err := users.Create(context.Background(), &User{Address: "0xb", InviterID: &a.ID})
	if err != nil {
		t.Fatal(err)
	}
	c, err := users.Create(context.Background(), &User{Address: "0xc", InviterID: &b.ID, PaidAmount: decimal.RequireFromString("1000")})
	if err != nil {
		t.Fatal(err)
	}
	d, err := users.Create(context.Background(), &User{Address: "0xd", InviterID: &c.ID})
	if err != nil {
		t.Fatal(err)
	}
	got, err := FindActivatedRecommendAncestors(context.Background(), users, d.ID, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != c.ID || got[1] != a.ID {
		t.Fatalf("%v", got)
	}
}

func TestSettle_ManageSkipsInactiveWalksUp(t *testing.T) {
	users := newMemUsers()
	i3, err := users.Create(context.Background(), &User{Address: "0xi3"})
	if err != nil {
		t.Fatal(err)
	}
	i2, err := users.Create(context.Background(), &User{Address: "0xi2", InviterID: &i3.ID})
	if err != nil {
		t.Fatal(err)
	}
	i1, err := users.Create(context.Background(), &User{Address: "0xi1", InviterID: &i2.ID})
	if err != nil {
		t.Fatal(err)
	}
	u, err := users.Create(context.Background(), &User{
		Address: "0xu", InviterID: &i1.ID, CapEffective: decimal.RequireFromString("600"),
	})
	if err != nil {
		t.Fatal(err)
	}
	place, orders := setupMatchTree(t, users, u)
	mustPay(t, orders, i3.ID, "1000", "2026-09-10")
	led := &memLedger{}
	uc := newSettleUCMatch(users, &memPackages{rows: seedCapPackages()}, &memSettleRuns{}, orders, led, place, newMemMatch(), true, shanghaiNoon("2026-09-11"))
	res, err := uc.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.ManageCount != 1 {
		t.Fatalf("manage=%d", res.ManageCount)
	}
	g1, _ := users.FindByID(context.Background(), i1.ID)
	g2, _ := users.FindByID(context.Background(), i2.ID)
	g3, _ := users.FindByID(context.Background(), i3.ID)
	if !g1.LockBalance.IsZero() || !g2.LockBalance.IsZero() {
		t.Fatalf("skipped inactive i1=%s i2=%s", g1.LockBalance, g2.LockBalance)
	}
	if !g3.AvailableBalance.Equal(decimal.RequireFromString("2")) {
		t.Fatalf("activated upper got %s", g3.AvailableBalance)
	}
}

func TestOrderPaid_InstantDirectMatchManage(t *testing.T) {
	users := newMemUsers()
	inv, err := users.Create(context.Background(), &User{Address: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
	if err != nil {
		t.Fatal(err)
	}
	invID := inv.ID
	a, err := users.Create(context.Background(), &User{
		Address: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", InviterID: &invID, CapEffective: decimal.RequireFromString("600"),
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := users.Create(context.Background(), &User{Address: "0xcccccccccccccccccccccccccccccccccccccccc", InviterID: &invID})
	if err != nil {
		t.Fatal(err)
	}
	c, err := users.Create(context.Background(), &User{Address: "0xdddddddddddddddddddddddddddddddddddddddd"})
	if err != nil {
		t.Fatal(err)
	}
	place := newMemPlacements()
	puc := NewPlacementUseCase(users, place, nil)
	if _, err := puc.Place(context.Background(), b.ID, a.ID, SideLeft); err != nil {
		t.Fatal(err)
	}
	if _, err := puc.Place(context.Background(), c.ID, a.ID, SideRight); err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	led := &memLedger{}
	pkgs := &memPackages{rows: seedCapPackages()}
	settle := newSettleUCMatch(users, pkgs, &memSettleRuns{}, orders, led, place, newMemMatch(), true, shanghaiNoon("2026-09-11"))
	ouc := NewOrderUseCase(pkgs, orders, users, users, led)
	ouc.SetPaidHook(settle)

	ob, err := ouc.CreateOrder(context.Background(), b.ID, decimal.RequireFromString("1000"), 300)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ouc.MarkPaid(context.Background(), ob.ID); err != nil {
		t.Fatal(err)
	}
	gotA, _ := users.FindByID(context.Background(), a.ID)
	if !gotA.LockBalance.IsZero() || !gotA.AvailableBalance.IsZero() {
		t.Fatalf("no pair yet A avail=%s lock=%s", gotA.AvailableBalance, gotA.LockBalance)
	}

	oc, err := ouc.CreateOrder(context.Background(), c.ID, decimal.RequireFromString("1000"), 300)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ouc.MarkPaid(context.Background(), oc.ID); err != nil {
		t.Fatal(err)
	}
	gotA, _ = users.FindByID(context.Background(), a.ID)
	if !gotA.LockBalance.Equal(decimal.RequireFromString("50")) {
		t.Fatalf("instant match A lock=%s", gotA.LockBalance)
	}
	gotInv, _ := users.FindByID(context.Background(), inv.ID)
	if !gotInv.LockBalance.Equal(decimal.RequireFromString("50")) {
		t.Fatalf("instant direct inv lock=%s", gotInv.LockBalance)
	}
}

func TestOrderStaticReleases_TodayLedgerAndPending(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xrel"})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	paidAt := shanghaiNoon("2026-09-10")
	o, err := orders.Create(context.Background(), &Order{
		UserID: u.ID, PackageID: 1, Amount: decimal.RequireFromString("12000"),
		Status: OrderPending, ReleaseDays: 300,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := orders.MarkPaid(context.Background(), o.ID, u.ID, o.Amount, paidAt); err != nil {
		t.Fatal(err)
	}
	got, err := orders.FindByID(context.Background(), o.ID)
	if err != nil {
		t.Fatal(err)
	}
	today := shanghaiNoon("2026-09-13")
	oid := o.ID
	led := &memLedger{rows: []*LedgerEntry{
		{UserID: u.ID, OrderID: &oid, EntryType: LedgerStatic, Amount: decimal.RequireFromString("33.33333000"), SettleDate: &today},
		{UserID: u.ID, OrderID: &oid, EntryType: LedgerStaticIspay, Amount: decimal.RequireFromString("0.01666667"), SettleDate: &today},
	}}
	uc := newSettleUCFull(users, &memPackages{}, &memSettleRuns{}, orders, led, &memConfigs{min: "10"}, false, today)
	m, err := uc.OrderStaticReleases(context.Background(), []*Order{got})
	if err != nil {
		t.Fatal(err)
	}
	rel := m[o.ID]
	if !rel.TodayUSDT.Equal(decimal.RequireFromString("33.33333000")) || !rel.TodayIspay.Equal(decimal.RequireFromString("0.01666667")) {
		t.Fatalf("today usdt=%s ispay=%s", rel.TodayUSDT, rel.TodayIspay)
	}
	if !rel.ReleasedUSDT.Equal(decimal.RequireFromString("33.33333000")) || !rel.ReleasedIspay.Equal(decimal.RequireFromString("0.01666667")) {
		t.Fatalf("released usdt=%s ispay=%s", rel.ReleasedUSDT, rel.ReleasedIspay)
	}
	if !rel.PendingUSDT.IsPositive() || !rel.PendingIspay.IsPositive() {
		t.Fatalf("pending usdt=%s ispay=%s", rel.PendingUSDT, rel.PendingIspay)
	}
	if rel.SettleDate != "2026-09-13" {
		t.Fatalf("settle_date=%s", rel.SettleDate)
	}
}
