package biz

import (
	"context"
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

func (m *memLedger) ListPaged(_ context.Context, address, entryType string, page, pageSize int) ([]*LedgerEntry, int, error) {
	_ = address
	var filtered []*LedgerEntry
	for _, e := range m.rows {
		if entryType != "" && e.EntryType != entryType {
			continue
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

func (m *memLedger) ExistsByOrderAndType(_ context.Context, orderID uint64, entryType string) (bool, error) {
	for _, e := range m.rows {
		if e.OrderID != nil && *e.OrderID == orderID && e.EntryType == entryType {
			return true, nil
		}
	}
	return false, nil
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
		{"2", "", false},
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
	uc := NewLedgerUseCase(repo)
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
		if it.Amount != "10.00000000" && it.Amount != "3.00000000" && it.Amount != "2.00000000" {
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
	uc := NewLedgerUseCase(repo)
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
	}})
	page, err := uc.ListUserRewards(context.Background(), 1, "2", 1)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 0 || len(page.Items) != 0 {
		t.Fatalf("%+v", page)
	}
}

func TestListUserRewards_EmptyWhenNoEngine(t *testing.T) {
	uc := NewLedgerUseCase(&memLedger{})
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
	uc := NewLedgerUseCase(&memLedger{rows: rows})
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
	uc := NewLedgerUseCase(repo)
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
