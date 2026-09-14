package biz

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

type memLedger struct {
	rows []*LedgerEntry
}

func (m *memLedger) Create(_ context.Context, e *LedgerEntry) error {
	cp := *e
	cp.ID = uint64(len(m.rows) + 1)
	m.rows = append(m.rows, &cp)
	return nil
}

func (m *memLedger) ListByUser(_ context.Context, userID uint64, from, to time.Time) ([]*LedgerEntry, error) {
	var out []*LedgerEntry
	for _, e := range m.rows {
		if e.UserID != userID {
			continue
		}
		if e.CreatedAt.Before(from) || !e.CreatedAt.Before(to) {
			continue
		}
		cp := *e
		out = append(out, &cp)
	}
	return out, nil
}

func (m *memLedger) ListPaged(_ context.Context, address string, entryTypes []string, page, pageSize int) ([]*LedgerEntry, int, error) {
	_ = address
	allow := map[string]struct{}{}
	for _, t := range entryTypes {
		allow[t] = struct{}{}
	}
	var filtered []*LedgerEntry
	for _, e := range m.rows {
		if len(allow) > 0 {
			if _, ok := allow[e.EntryType]; !ok {
				continue
			}
		}
		cp := *e
		filtered = append(filtered, &cp)
	}
	total := len(filtered)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultRewardPageSize
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []*LedgerEntry{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return filtered[start:end], total, nil
}

func (m *memLedger) FindMatching(_ context.Context, userID uint64, entryType string, orderID *uint64, settleDate *time.Time, remark string) (*LedgerEntry, error) {
	day := ""
	if settleDate != nil {
		day = settleDate.Format("2006-01-02")
	}
	for _, e := range m.rows {
		if e.UserID != userID || e.EntryType != entryType || e.Remark != remark {
			continue
		}
		if orderID != nil {
			if e.OrderID == nil || *e.OrderID != *orderID {
				continue
			}
		} else if e.OrderID != nil {
			continue
		}
		if day != "" {
			if e.SettleDate == nil || e.SettleDate.Format("2006-01-02") != day {
				continue
			}
		}
		cp := *e
		return &cp, nil
	}
	return nil, nil
}

func (m *memLedger) ExistsByOrderAndType(_ context.Context, orderID uint64, entryType string) (bool, error) {
	for _, e := range m.rows {
		if e.OrderID != nil && *e.OrderID == orderID && e.EntryType == entryType {
			return true, nil
		}
	}
	return false, nil
}

func (m *memLedger) ExistsByOrderTypeAndDate(_ context.Context, orderID uint64, entryType string, settleDate time.Time) (bool, error) {
	day := settleDate.Format("2006-01-02")
	for _, e := range m.rows {
		if e.OrderID == nil || *e.OrderID != orderID || e.EntryType != entryType || e.SettleDate == nil {
			continue
		}
		if e.SettleDate.Format("2006-01-02") == day {
			return true, nil
		}
	}
	return false, nil
}

func (m *memLedger) CountByOrderAndType(_ context.Context, orderID uint64, entryType string) (int, error) {
	n := 0
	for _, e := range m.rows {
		if e.OrderID != nil && *e.OrderID == orderID && e.EntryType == entryType {
			n++
		}
	}
	return n, nil
}

func (m *memLedger) ListByOrderIDsAndTypes(_ context.Context, orderIDs []uint64, entryTypes []string) ([]*LedgerEntry, error) {
	if len(orderIDs) == 0 || len(entryTypes) == 0 {
		return nil, nil
	}
	allowID := map[uint64]struct{}{}
	for _, id := range orderIDs {
		allowID[id] = struct{}{}
	}
	allowType := map[string]struct{}{}
	for _, t := range entryTypes {
		allowType[t] = struct{}{}
	}
	var out []*LedgerEntry
	for _, e := range m.rows {
		if e.OrderID == nil {
			continue
		}
		if _, ok := allowID[*e.OrderID]; !ok {
			continue
		}
		if _, ok := allowType[e.EntryType]; !ok {
			continue
		}
		cp := *e
		out = append(out, &cp)
	}
	return out, nil
}

func (m *memLedger) ExistsByUserTypeAndDate(_ context.Context, userID uint64, entryType string, settleDate time.Time) (bool, error) {
	day := settleDate.Format("2006-01-02")
	for _, e := range m.rows {
		if e.UserID != userID || e.EntryType != entryType || e.SettleDate == nil {
			continue
		}
		if e.SettleDate.Format("2006-01-02") == day {
			return true, nil
		}
	}
	return false, nil
}

func (m *memLedger) ExistsByUserTypeDateRemark(_ context.Context, userID uint64, entryType string, settleDate time.Time, remark string) (bool, error) {
	day := settleDate.Format("2006-01-02")
	for _, e := range m.rows {
		if e.UserID != userID || e.EntryType != entryType || e.Remark != remark || e.SettleDate == nil {
			continue
		}
		if e.SettleDate.Format("2006-01-02") == day {
			return true, nil
		}
	}
	return false, nil
}

func (m *memLedger) ListByTypeAndDate(_ context.Context, entryType string, settleDate time.Time) ([]*LedgerEntry, error) {
	day := settleDate.Format("2006-01-02")
	var out []*LedgerEntry
	for _, e := range m.rows {
		if e.EntryType != entryType || e.SettleDate == nil {
			continue
		}
		if e.SettleDate.Format("2006-01-02") != day {
			continue
		}
		cp := *e
		out = append(out, &cp)
	}
	return out, nil
}

func TestReqTypeToEntryType(t *testing.T) {
	cases := []struct {
		in     string
		want   string
		wantOK bool
	}{
		{"", "", true},
		{"1", "", true},
		{"3", LedgerDirect, true},
		{"4", LedgerMatch, true},
		{"5", LedgerManage, true},
		{"2", LedgerStatic, true},
		{"99", "", false},
	}
	for _, tc := range cases {
		got, ok := ReqTypeToEntryType(tc.in)
		if ok != tc.wantOK || got != tc.want {
			t.Fatalf("reqType=%q got=(%q,%v) want=(%q,%v)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestListUserRewards_DefaultFiltersEarningsOnly(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	repo := &memLedger{rows: []*LedgerEntry{
		{ID: 1, UserID: 1, EntryType: LedgerDirect, Amount: decimal.RequireFromString("10"), CreatedAt: now},
		{ID: 2, UserID: 1, EntryType: LedgerWithdraw, Amount: decimal.RequireFromString("5"), CreatedAt: now},
		{ID: 3, UserID: 1, EntryType: LedgerMatch, Amount: decimal.RequireFromString("3"), CreatedAt: now},
		{ID: 4, UserID: 1, EntryType: LedgerFreeze, Amount: decimal.RequireFromString("1"), CreatedAt: now},
		{ID: 5, UserID: 1, EntryType: LedgerManage, Amount: decimal.RequireFromString("2"), CreatedAt: now},
		{ID: 6, UserID: 2, EntryType: LedgerDirect, Amount: decimal.RequireFromString("9"), CreatedAt: now},
	}}
	uc := NewLedgerUseCase(repo, nil, nil)
	uc.now = func() time.Time { return now }

	page, err := uc.ListUserRewards(context.Background(), 1, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3 {
		t.Fatalf("total=%d want 3", page.Total)
	}
	if len(page.Items) != 3 {
		t.Fatalf("items=%d", len(page.Items))
	}
	for _, it := range page.Items {
		if !IsRewardEntry(ReasonToEntryType(it.Reason)) && it.Reason != "direct" && it.Reason != "match" && it.Reason != "manage" {
			t.Fatalf("unexpected reason %s", it.Reason)
		}
		if it.Address != "" || it.Num != "" {
			t.Fatalf("address/num should be empty")
		}
		if it.Amount != "10" && it.Amount != "3" && it.Amount != "2" {
			t.Fatalf("amount=%s", it.Amount)
		}
	}
}

func TestListUserRewards_ReqTypeSingle(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	repo := &memLedger{rows: []*LedgerEntry{
		{ID: 1, UserID: 1, EntryType: LedgerDirect, Amount: decimal.RequireFromString("10"), CreatedAt: now},
		{ID: 2, UserID: 1, EntryType: LedgerMatch, Amount: decimal.RequireFromString("3"), CreatedAt: now},
	}}
	uc := NewLedgerUseCase(repo, nil, nil)
	uc.now = func() time.Time { return now }

	page, err := uc.ListUserRewards(context.Background(), 1, "4", 1)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Reason != "match" {
		t.Fatalf("%+v", page)
	}
}

func TestListUserRewards_UnknownReqTypeEmpty(t *testing.T) {
	uc := NewLedgerUseCase(&memLedger{rows: []*LedgerEntry{
		{ID: 1, UserID: 1, EntryType: LedgerDirect, Amount: decimal.RequireFromString("1"), CreatedAt: time.Now()},
	}}, nil, nil)
	page, err := uc.ListUserRewards(context.Background(), 1, "2", 1)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 0 || len(page.Items) != 0 {
		t.Fatalf("%+v", page)
	}
}

func TestListUserRewards_EmptyWhenNoEngine(t *testing.T) {
	uc := NewLedgerUseCase(&memLedger{}, nil, nil)
	page, err := uc.ListUserRewards(context.Background(), 1, "1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 0 || page.Items == nil || len(page.Items) != 0 {
		t.Fatalf("%+v", page)
	}
}

func TestListUserRewards_Pagination(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	var rows []*LedgerEntry
	for i := 0; i < 12; i++ {
		rows = append(rows, &LedgerEntry{
			ID: uint64(i + 1), UserID: 1, EntryType: LedgerDirect,
			Amount: decimal.NewFromInt(int64(i + 1)), CreatedAt: now,
		})
	}
	uc := NewLedgerUseCase(&memLedger{rows: rows}, nil, nil)
	uc.now = func() time.Time { return now }
	p1, err := uc.ListUserRewards(context.Background(), 1, "3", 1)
	if err != nil {
		t.Fatal(err)
	}
	if p1.Total != 12 || len(p1.Items) != 10 {
		t.Fatalf("page1 total=%d n=%d", p1.Total, len(p1.Items))
	}
	p2, err := uc.ListUserRewards(context.Background(), 1, "3", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(p2.Items) != 2 {
		t.Fatalf("page2 n=%d", len(p2.Items))
	}
}

func TestListAdminRewards_ReasonFilter(t *testing.T) {
	now := time.Now()
	repo := &memLedger{rows: []*LedgerEntry{
		{ID: 1, UserID: 1, EntryType: LedgerDirect, Amount: decimal.RequireFromString("1"), Remark: "r1", CreatedAt: now},
		{ID: 2, UserID: 1, EntryType: LedgerWithdraw, Amount: decimal.RequireFromString("2"), Remark: "r2", CreatedAt: now},
	}}
	uc := NewLedgerUseCase(repo, nil, nil)
	page, err := uc.ListAdminRewards(context.Background(), "", "direct", 1)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].Remark != "r1" {
		t.Fatalf("%+v", page)
	}
	all, err := uc.ListAdminRewards(context.Background(), "", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if all.Total != 2 {
		t.Fatalf("all total=%d", all.Total)
	}
}

func TestListAdminRewards_RewardPairAndCategory(t *testing.T) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	day := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	oid := uint64(8)
	repo := &memLedger{rows: []*LedgerEntry{
		{ID: 1, UserID: 2, OrderID: &oid, EntryType: LedgerStatic, Amount: decimal.RequireFromString("1.2"), BalanceKind: BalanceAvailable, SettleDate: &day, Remark: "static order=8 days=300 day=1", Address: "0xabc", CreatedAt: now},
		{ID: 2, UserID: 2, OrderID: &oid, EntryType: LedgerStaticIspay, Amount: decimal.RequireFromString("0.0006"), BalanceKind: BalanceIspay, SettleDate: &day, Remark: "static order=8 days=300 day=1", Address: "0xabc", CreatedAt: now},
		{ID: 3, UserID: 2, EntryType: LedgerDirect, Amount: decimal.RequireFromString("5"), BalanceKind: BalanceLock, SettleDate: &day, Remark: "direct order=3 rate=0.1", Address: "0xabc", CreatedAt: now},
		{ID: 4, UserID: 2, EntryType: LedgerWithdraw, Amount: decimal.RequireFromString("9"), CreatedAt: now},
	}}
	uc := NewLedgerUseCase(repo, nil, nil)
	page, err := uc.ListAdminRewards(context.Background(), "", "reward", 1)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("reward page %+v", page)
	}
	static := page.Items[0]
	if static.Category != "静态收益" || static.Reason != "static" || static.Amount != "1.2" || static.AmountTwo != "0.0006" {
		t.Fatalf("static %+v", static)
	}
	if static.Detail != "订单#8 · 300天档 · 第1天" || static.Address != "0xabc" || static.BalanceName != "可提U" {
		t.Fatalf("static detail %+v", static)
	}
	direct := page.Items[1]
	if direct.Category != "动态收益" || direct.AmountTwo != "0" || !strings.Contains(direct.Detail, "费率 0.1") {
		t.Fatalf("direct %+v", direct)
	}
	onlyStatic, err := uc.ListAdminRewards(context.Background(), "", "static", 1)
	if err != nil {
		t.Fatal(err)
	}
	if onlyStatic.Total != 1 || onlyStatic.Items[0].Reason != "static" {
		t.Fatalf("static filter %+v", onlyStatic)
	}
}

func TestListAdminRewards_OrderSourceAmount(t *testing.T) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	day := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	orders := newMemOrders(nil)
	o, err := orders.Create(context.Background(), &Order{
		UserID: 2, PackageID: 1, Amount: decimal.RequireFromString("1000"),
		TitleSnapshot: "牙刷挖矿", Status: OrderPaid,
	})
	if err != nil {
		t.Fatal(err)
	}
	oid := o.ID
	users := newMemUsers()
	buyer, err := users.Create(context.Background(), &User{Address: "0xbuyer"})
	if err != nil {
		t.Fatal(err)
	}
	o.UserID = buyer.ID
	orders.byID[o.ID].UserID = buyer.ID
	uc := NewLedgerUseCase(&memLedger{rows: []*LedgerEntry{
		{ID: 1, UserID: 2, OrderID: &oid, EntryType: LedgerStatic, Amount: decimal.RequireFromString("2.77778"), BalanceKind: BalanceAvailable, SettleDate: &day, Address: "0xabc", CreatedAt: now},
	}}, orders, users)
	page, err := uc.ListAdminRewards(context.Background(), "", "static", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("%+v", page)
	}
	it := page.Items[0]
	if it.OrderID != strconv.FormatUint(oid, 10) || it.OrderAmount != "1000" || it.OrderTitle != "牙刷挖矿" {
		t.Fatalf("source %+v", it)
	}
	if it.OrderNo != FormatOrderNo(oid) || it.OrderSource != FormatOrderSource(it.OrderNo, it.OrderAmount) {
		t.Fatalf("order source %+v", it)
	}
	if it.SourceAddress != "0xbuyer" {
		t.Fatalf("source address %+v", it)
	}
}

func TestListAdminRewards_ManageOrderKeySource(t *testing.T) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	day := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	users := newMemUsers()
	buyer, err := users.Create(context.Background(), &User{Address: "0xorderbuyer"})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(nil)
	o, err := orders.Create(context.Background(), &Order{
		UserID: buyer.ID, PackageID: 1, Amount: decimal.RequireFromString("36000"),
		TitleSnapshot: "组合", Status: OrderPaid,
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewLedgerUseCase(&memLedger{rows: []*LedgerEntry{
		{ID: 1, UserID: 9, EntryType: LedgerManage, Amount: decimal.RequireFromString("1"), BalanceKind: BalanceAvailable, SettleDate: &day, Remark: fmt.Sprintf("manage gen=1 source=99 key=order=%d", o.ID), Address: "0xrecv", CreatedAt: now},
	}}, orders, users)
	page, err := uc.ListAdminRewards(context.Background(), "", "manage", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("%+v", page)
	}
	it := page.Items[0]
	if it.OrderSource != FormatOrderSource(o.DisplayNo(), "36000") || it.SourceAddress != "0xorderbuyer" {
		t.Fatalf("manage order source %+v", it)
	}
}

func TestListAdminRewards_ManageSourceAddress(t *testing.T) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	day := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	users := newMemUsers()
	src, err := users.Create(context.Background(), &User{Address: "0xsource"})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewLedgerUseCase(&memLedger{rows: []*LedgerEntry{
		{ID: 1, UserID: 9, EntryType: LedgerManage, Amount: decimal.RequireFromString("1"), BalanceKind: BalanceAvailable, SettleDate: &day, Remark: fmt.Sprintf("manage gen=1 source=%d key=leftover", src.ID), Address: "0xrecv", CreatedAt: now},
	}}, nil, users)
	page, err := uc.ListAdminRewards(context.Background(), "", "manage", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].SourceAddress != "0xsource" {
		t.Fatalf("manage source %+v", page)
	}
}
