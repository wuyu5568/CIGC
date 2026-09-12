package biz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

type memWithdraws struct {
	byID map[uint64]*Withdraw
	next uint64
}

func newMemWithdraws() *memWithdraws {
	return &memWithdraws{byID: map[uint64]*Withdraw{}, next: 1}
}

func (m *memWithdraws) Create(_ context.Context, w *Withdraw) (*Withdraw, error) {
	cp := *w
	cp.ID = m.next
	m.next++
	cp.CreatedAt = time.Now()
	m.byID[cp.ID] = &cp
	out := cp
	return &out, nil
}

func (m *memWithdraws) FindByID(_ context.Context, id uint64) (*Withdraw, error) {
	w, ok := m.byID[id]
	if !ok {
		return nil, ErrWithdrawNotFound
	}
	cp := *w
	return &cp, nil
}

func (m *memWithdraws) CasStatus(_ context.Context, id uint64, from, to, remark string, reviewedAt *time.Time) error {
	w, ok := m.byID[id]
	if !ok {
		return ErrWithdrawNotFound
	}
	if w.Status != from {
		return ErrWithdrawConflict
	}
	w.Status = to
	w.Remark = remark
	w.ReviewedAt = reviewedAt
	return nil
}

func (m *memWithdraws) ListByUser(_ context.Context, userID uint64) ([]*Withdraw, error) {
	var out []*Withdraw
	for _, w := range m.byID {
		if w.UserID == userID {
			cp := *w
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (m *memWithdraws) ListAdmin(_ context.Context, address, status string, page, pageSize int) ([]*AdminWithdrawRow, int, error) {
	_ = address
	var filtered []*AdminWithdrawRow
	for _, w := range m.byID {
		if status != "" && w.Status != status {
			continue
		}
		cp := *w
		filtered = append(filtered, &AdminWithdrawRow{Withdraw: cp})
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
		return []*AdminWithdrawRow{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return filtered[start:end], total, nil
}

type memConfigs struct {
	min        string
	directRate string
	matchRate  string
}

func (m *memConfigs) GetValue(_ context.Context, key string) (string, error) {
	switch key {
	case ConfigMinWithdraw:
		if m.min != "" {
			return m.min, nil
		}
	case ConfigDirectRate:
		if m.directRate != "" {
			return m.directRate, nil
		}
	case ConfigMatchRate:
		if m.matchRate != "" {
			return m.matchRate, nil
		}
	}
	return "", nil
}

func newWithdrawUC(users *memUsers, led *memLedger, wds *memWithdraws, min string) *WithdrawUseCase {
	return NewWithdrawUseCase(users, users, led, wds, &memConfigs{min: min}, NopTx{})
}

func TestCreateWithdraw_FreezesAvailable(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("100"),
	})
	if err != nil {
		t.Fatal(err)
	}
	led := &memLedger{}
	wds := newMemWithdraws()
	uc := newWithdrawUC(users, led, wds, "10")

	wd, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("20"))
	if err != nil {
		t.Fatal(err)
	}
	if wd.Status != WithdrawPending || !wd.FeeAmount.IsZero() || !wd.CreditedAmount.Equal(decimal.RequireFromString("20")) {
		t.Fatalf("%+v", wd)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.AvailableBalance.Equal(decimal.RequireFromString("80")) || !got.FrozenBalance.Equal(decimal.RequireFromString("20")) {
		t.Fatalf("avail=%s frozen=%s", got.AvailableBalance, got.FrozenBalance)
	}
	if len(led.rows) != 2 {
		t.Fatalf("ledger n=%d", len(led.rows))
	}
}

func TestCreateWithdraw_BelowMinAndInsufficient(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("15"),
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := newWithdrawUC(users, &memLedger{}, newMemWithdraws(), "10")
	if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("9")); !errors.Is(err, ErrWithdrawBelowMin) {
		t.Fatalf("min: %v", err)
	}
	if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("16")); !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("bal: %v", err)
	}
}

func TestCreateWithdraw_DisabledUser(t *testing.T) {
	users := newMemUsers()
	now := time.Now()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("100"),
		DisabledAt:       &now,
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := newWithdrawUC(users, &memLedger{}, newMemWithdraws(), "10")
	if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("10")); !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("got %v", err)
	}
}

func TestWithdrawPassKeepsFrozen(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("50"),
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := newWithdrawUC(users, &memLedger{}, newMemWithdraws(), "10")
	wd, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("10"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := uc.Pass(context.Background(), wd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != WithdrawRewarded {
		t.Fatalf("status %s", got.Status)
	}
	u2, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !u2.FrozenBalance.Equal(decimal.RequireFromString("10")) {
		t.Fatalf("frozen=%s", u2.FrozenBalance)
	}
	if _, err := uc.Pass(context.Background(), wd.ID); !errors.Is(err, ErrWithdrawConflict) {
		t.Fatalf("double pass: %v", err)
	}
}

func TestWithdrawRejectUnfreezes(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("50"),
	})
	if err != nil {
		t.Fatal(err)
	}
	led := &memLedger{}
	uc := newWithdrawUC(users, led, newMemWithdraws(), "10")
	wd, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("10"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := uc.Reject(context.Background(), wd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != WithdrawRejected {
		t.Fatalf("status %s", got.Status)
	}
	u2, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !u2.AvailableBalance.Equal(decimal.RequireFromString("50")) || !u2.FrozenBalance.IsZero() {
		t.Fatalf("avail=%s frozen=%s", u2.AvailableBalance, u2.FrozenBalance)
	}
	unfreeze := 0
	for _, e := range led.rows {
		if e.EntryType == LedgerUnfreeze {
			unfreeze++
		}
	}
	if unfreeze != 2 {
		t.Fatalf("unfreeze entries=%d", unfreeze)
	}
}

func TestListUserWithdrawPagination(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("200"),
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := newWithdrawUC(users, &memLedger{}, newMemWithdraws(), "10")
	for i := 0; i < 11; i++ {
		if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("10")); err != nil {
			t.Fatal(err)
		}
	}
	p1, err := uc.ListUser(context.Background(), u.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if p1.Total != 11 || len(p1.Items) != 10 {
		t.Fatalf("p1 total=%d n=%d", p1.Total, len(p1.Items))
	}
}
