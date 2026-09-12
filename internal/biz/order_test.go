package biz

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

type memPackages struct {
	rows []*Package
}

func (m *memPackages) ListEnabled(_ context.Context) ([]*Package, error) {
	out := make([]*Package, 0, len(m.rows))
	for _, p := range m.rows {
		if p.Enabled {
			cp := *p
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (m *memPackages) ListAll(_ context.Context) ([]*Package, error) {
	out := make([]*Package, 0, len(m.rows))
	for _, p := range m.rows {
		cp := *p
		out = append(out, &cp)
	}
	return out, nil
}

func (m *memPackages) FindByAmount(_ context.Context, amount decimal.Decimal) (*Package, error) {
	for _, p := range m.rows {
		if p.Amount.Equal(amount) {
			cp := *p
			return &cp, nil
		}
	}
	return nil, ErrPackageNotFound
}

func (m *memPackages) FindByID(_ context.Context, id uint64) (*Package, error) {
	for _, p := range m.rows {
		if p.ID == id {
			cp := *p
			return &cp, nil
		}
	}
	return nil, ErrPackageNotFound
}

type memOrders struct {
	byID  map[uint64]*Order
	next  uint64
	users *memUsers
}

func newMemOrders(users *memUsers) *memOrders {
	return &memOrders{byID: map[uint64]*Order{}, next: 1, users: users}
}

func (m *memOrders) Create(_ context.Context, o *Order) (*Order, error) {
	cp := *o
	cp.ID = m.next
	m.next++
	cp.CreatedAt = time.Now()
	m.byID[cp.ID] = &cp
	out := cp
	return &out, nil
}

func (m *memOrders) FindByID(_ context.Context, id uint64) (*Order, error) {
	o, ok := m.byID[id]
	if !ok {
		return nil, ErrOrderNotFound
	}
	cp := *o
	return &cp, nil
}

func (m *memOrders) ListByUser(_ context.Context, userID uint64) ([]*Order, error) {
	var out []*Order
	for _, o := range m.byID {
		if o.UserID == userID {
			cp := *o
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (m *memOrders) MarkPaid(_ context.Context, id uint64, userID uint64, amount decimal.Decimal, paidAt time.Time) error {
	o, ok := m.byID[id]
	if !ok {
		return ErrOrderNotFound
	}
	if o.Status != OrderPending {
		return ErrOrderConflict
	}
	o.Status = OrderPaid
	o.PaidAt = &paidAt
	if m.users != nil {
		return m.users.AddPaidAmount(context.Background(), userID, amount)
	}
	return nil
}

func (m *memOrders) ListAdmin(_ context.Context, address, status string, page, pageSize int) ([]*AdminOrderRow, int, error) {
	var filtered []*AdminOrderRow
	for _, o := range m.byID {
		if status != "" && o.Status != status {
			continue
		}
		addr := ""
		if m.users != nil {
			if u, err := m.users.FindByID(context.Background(), o.UserID); err == nil {
				addr = u.Address
			}
		}
		if address != "" && !stringContains(addr, address) {
			continue
		}
		cp := *o
		filtered = append(filtered, &AdminOrderRow{Order: cp, Address: addr})
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
		return []*AdminOrderRow{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return filtered[start:end], total, nil
}

func (m *memOrders) ListPaidBetween(_ context.Context, from, to time.Time) ([]*Order, error) {
	var out []*Order
	for _, o := range m.byID {
		if o.Status != OrderPaid || o.PaidAt == nil {
			continue
		}
		if o.PaidAt.Before(from) || !o.PaidAt.Before(to) {
			continue
		}
		cp := *o
		out = append(out, &cp)
	}
	return out, nil
}

func TestCreateOrder_AndMarkPaid(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xabc"})
	if err != nil {
		t.Fatal(err)
	}
	pkgs := &memPackages{rows: []*Package{{
		ID: 1, Amount: decimal.RequireFromString("1000"), Title: "t", Enabled: true,
	}}}
	orders := newMemOrders(users)
	uc := NewOrderUseCase(pkgs, orders, users)

	o, err := uc.CreateOrder(context.Background(), u.ID, decimal.RequireFromString("1000"))
	if err != nil {
		t.Fatal(err)
	}
	if o.Status != OrderPending {
		t.Fatalf("status %s", o.Status)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.PaidAmount.IsZero() {
		t.Fatalf("unpaid should not increase paid_amount: %s", got.PaidAmount)
	}

	paid, err := uc.MarkPaid(context.Background(), o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if paid.Status != OrderPaid {
		t.Fatalf("status %s", paid.Status)
	}
	got, err = users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.PaidAmount.Equal(decimal.RequireFromString("1000")) {
		t.Fatalf("paid_amount=%s", got.PaidAmount)
	}
	if _, err := uc.MarkPaid(context.Background(), o.ID); err != ErrOrderConflict {
		t.Fatalf("double pay: %v", err)
	}
}

func TestCreateOrder_UnknownAmount(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xabc"})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewOrderUseCase(&memPackages{}, newMemOrders(users), users)
	_, err = uc.CreateOrder(context.Background(), u.ID, decimal.RequireFromString("999"))
	if err != ErrPackageNotFound {
		t.Fatalf("got %v", err)
	}
}
