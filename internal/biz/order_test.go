package biz

import (
	"context"
	"strings"
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

func (m *memPackages) FindByAmount(_ context.Context, amount decimal.Decimal, days int) (*Package, error) {
	if !ValidReleaseDays(days) {
		days = ReleaseDays300
	}
	for _, p := range m.rows {
		if p.Amount.Equal(amount) && PackageReleaseDays(p) == days {
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

func (m *memPackages) Update(_ context.Context, p *Package) (*Package, error) {
	for i, row := range m.rows {
		if row.ID == p.ID {
			cp := *p
			m.rows[i] = &cp
			out := cp
			return &out, nil
		}
	}
	return nil, ErrPackageNotFound
}

func (m *memPackages) UpdateSortOrder(_ context.Context, id uint64, sort int) error {
	for _, row := range m.rows {
		if row.ID == id {
			row.SortOrder = sort
			return nil
		}
	}
	return ErrPackageNotFound
}

func (m *memPackages) Create(_ context.Context, p *Package) (*Package, error) {
	cp := *p
	var maxID uint64
	for _, row := range m.rows {
		if row.ID > maxID {
			maxID = row.ID
		}
	}
	cp.ID = maxID + 1
	m.rows = append(m.rows, &cp)
	out := cp
	return &out, nil
}

func (m *memPackages) Delete(_ context.Context, id uint64) error {
	for i, row := range m.rows {
		if row.ID == id {
			m.rows = append(m.rows[:i], m.rows[i+1:]...)
			return nil
		}
	}
	return ErrPackageNotFound
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
	if strings.TrimSpace(cp.OrderNo) == "" {
		cp.OrderNo = FormatOrderNo(cp.ID)
	}
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

func (m *memOrders) MaxPaidAmountByUser(_ context.Context) (map[uint64]decimal.Decimal, error) {
	out := map[uint64]decimal.Decimal{}
	for _, o := range m.byID {
		if o.Status != OrderPaid {
			continue
		}
		if cur, ok := out[o.UserID]; !ok || o.Amount.GreaterThan(cur) {
			out[o.UserID] = o.Amount
		}
	}
	return out, nil
}

func (m *memOrders) MarkPaid(_ context.Context, id uint64, userID uint64, amount decimal.Decimal, paidAt time.Time) error {
	return m.markPaid(id, userID, amount, paidAt, nil, 0)
}

func (m *memOrders) MarkPaidWithChain(_ context.Context, id uint64, userID uint64, amount decimal.Decimal, paidAt time.Time, txHash string, logIndex int) error {
	h := strings.ToLower(strings.TrimSpace(txHash))
	return m.markPaid(id, userID, amount, paidAt, &h, logIndex)
}

func (m *memOrders) markPaid(id uint64, userID uint64, amount decimal.Decimal, paidAt time.Time, txHash *string, logIndex int) error {
	o, ok := m.byID[id]
	if !ok {
		return ErrOrderNotFound
	}
	if o.Status != OrderPending {
		return ErrOrderConflict
	}
	if txHash != nil && *txHash != "" {
		for _, other := range m.byID {
			if other.TxHash != nil && *other.TxHash == *txHash && other.LogIndex == logIndex {
				return ErrOrderConflict
			}
		}
		o.TxHash = txHash
		o.LogIndex = logIndex
	}
	o.Status = OrderPaid
	o.PaidAt = &paidAt
	if m.users != nil {
		return m.users.AddPaidAmount(context.Background(), userID, amount)
	}
	return nil
}

func (m *memOrders) FindOldestPendingByUserAmount(_ context.Context, userID uint64, amount decimal.Decimal) (*Order, error) {
	var best *Order
	for _, o := range m.byID {
		if o.UserID != userID || o.Status != OrderPending || !o.Amount.Equal(amount) {
			continue
		}
		if best == nil || o.ID < best.ID {
			cp := *o
			best = &cp
		}
	}
	return best, nil
}

func (m *memOrders) FindByTxEvent(_ context.Context, txHash string, logIndex int) (*Order, error) {
	txHash = strings.ToLower(strings.TrimSpace(txHash))
	for _, o := range m.byID {
		if o.TxHash != nil && *o.TxHash == txHash && o.LogIndex == logIndex {
			cp := *o
			return &cp, nil
		}
	}
	return nil, nil
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

func (m *memOrders) ListPaidBefore(_ context.Context, to time.Time) ([]*Order, error) {
	var out []*Order
	for _, o := range m.byID {
		if o.Status != OrderPaid || o.PaidAt == nil || !o.PaidAt.Before(to) {
			continue
		}
		cp := *o
		out = append(out, &cp)
	}
	return out, nil
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

func TestFilterPackagesByDays(t *testing.T) {
	pkgs := []*Package{
		{ID: 1, Title: "a", ReleaseDays: 300},
		{ID: 2, Title: "b", ReleaseDays: 600},
		{ID: 3, Title: "c", ReleaseDays: 750},
		{ID: 4, Title: "d", ReleaseDays: 0},
	}
	if n := FilterPackagesByDays(pkgs, 0); len(n) != 4 {
		t.Fatalf("all=%d", len(n))
	}
	got := FilterPackagesByDays(pkgs, 300)
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 4 {
		t.Fatalf("300=%+v", got)
	}
	got = FilterPackagesByDays(pkgs, 600)
	if len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("600=%+v", got)
	}
	if n := FilterPackagesByDays(pkgs, 750); len(n) != 1 || n[0].ID != 3 {
		t.Fatalf("750=%+v", n)
	}
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
	uc := NewOrderUseCase(pkgs, orders, users, users, &memLedger{})

	o, err := uc.CreateOrder(context.Background(), u.ID, decimal.RequireFromString("1000"), 300)
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
	uc := NewOrderUseCase(&memPackages{}, newMemOrders(users), users, users, &memLedger{})
	_, err = uc.CreateOrder(context.Background(), u.ID, decimal.RequireFromString("999"), 300)
	if err != ErrPackageNotFound {
		t.Fatalf("got %v", err)
	}
}

func TestBuyWithRecharge_PaysAndDeducts(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address: "0xabc", RechargeBalance: decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	pkgs := &memPackages{rows: []*Package{{
		ID: 1, Amount: decimal.RequireFromString("1000"), Title: "t", Enabled: true,
	}}}
	led := &memLedger{}
	uc := NewOrderUseCase(pkgs, newMemOrders(users), users, users, led)
	o, err := uc.BuyWithRecharge(context.Background(), u.ID, decimal.RequireFromString("1000"), 300)
	if err != nil {
		t.Fatal(err)
	}
	if o.Status != OrderPaid {
		t.Fatalf("status %s", o.Status)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.RechargeBalance.IsZero() || !got.PaidAmount.Equal(decimal.RequireFromString("1000")) {
		t.Fatalf("recharge=%s paid=%s", got.RechargeBalance, got.PaidAmount)
	}
	if len(led.rows) == 0 || led.rows[0].EntryType != LedgerRechargeBuy {
		t.Fatalf("ledger=%+v", led.rows)
	}
	if _, err := uc.BuyWithRecharge(context.Background(), u.ID, decimal.RequireFromString("1000"), 300); err != ErrInsufficientBalance {
		t.Fatalf("second buy: %v", err)
	}
}

func TestBuyWithRechargeGoods_UsesPackageAmount(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address: "0xabc", RechargeBalance: decimal.RequireFromString("2000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	pkgs := &memPackages{rows: []*Package{{
		ID: 9, Amount: decimal.RequireFromString("1500"), Title: "牙刷", GoodsDesc: "d", Enabled: true,
	}}}
	uc := NewOrderUseCase(pkgs, newMemOrders(users), users, users, &memLedger{})
	o, err := uc.BuyWithRechargeGoods(context.Background(), u.ID, 9, 750)
	if err != nil {
		t.Fatal(err)
	}
	if o.PackageID != 9 || !o.Amount.Equal(decimal.RequireFromString("1500")) || o.ReleaseDays != 750 {
		t.Fatalf("%+v", o)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.RechargeBalance.Equal(decimal.RequireFromString("500")) {
		t.Fatalf("recharge=%s", got.RechargeBalance)
	}
	if !got.CapEffective.Equal(decimal.RequireFromString("600")) {
		t.Fatalf("cap=%s", got.CapEffective)
	}
}

func TestBuyCartWithRecharge_SumsAndCaps(t *testing.T) {
	ResetRuntimeCapTiers()
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address: "0xabc", RechargeBalance: decimal.RequireFromString("5000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	ords := newMemOrders(users)
	uc := NewOrderUseCase(&memPackages{rows: []*Package{
		{ID: 1, Amount: decimal.RequireFromString("1000"), Title: "牙刷挖矿", Enabled: true},
		{ID: 3, Amount: decimal.RequireFromString("2000"), Title: "节点套餐", Enabled: true},
	}}, ords, users, users, &memLedger{})
	o, err := uc.BuyCartWithRecharge(context.Background(), u.ID, []CartItem{
		{GoodsID: 1, Qty: 2},
		{GoodsID: 1, Qty: 1},
		{GoodsID: 3, Qty: 1},
	}, 300)
	if err != nil {
		t.Fatal(err)
	}
	if o.Status != OrderPaid || !o.Amount.Equal(decimal.RequireFromString("5000")) {
		t.Fatalf("order %+v", o)
	}
	if o.PackageID != 3 {
		t.Fatalf("package_id=%d", o.PackageID)
	}
	if o.TitleSnapshot != "牙刷挖矿×3、节点套餐" {
		t.Fatalf("title=%s", o.TitleSnapshot)
	}
	list, err := ords.ListByUser(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("orders=%d", len(list))
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.RechargeBalance.IsZero() {
		t.Fatalf("recharge=%s", got.RechargeBalance)
	}
	if !got.CapEffective.Equal(decimal.RequireFromString("1800")) {
		t.Fatalf("cap=%s", got.CapEffective)
	}
}

func TestBuyCartWithRecharge_ThreeThousandCaps1800(t *testing.T) {
	ResetRuntimeCapTiers()
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address: "0xabc", RechargeBalance: decimal.RequireFromString("5000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	ords := newMemOrders(users)
	uc := NewOrderUseCase(&memPackages{rows: []*Package{
		{ID: 1, Amount: decimal.RequireFromString("1000"), Title: "牙刷挖矿", Enabled: true},
	}}, ords, users, users, &memLedger{})
	o, err := uc.BuyCartWithRecharge(context.Background(), u.ID, []CartItem{
		{GoodsID: 1, Qty: 3},
	}, 300)
	if err != nil {
		t.Fatal(err)
	}
	if !o.Amount.Equal(decimal.RequireFromString("3000")) || o.Status != OrderPaid {
		t.Fatalf("%+v", o)
	}
	if o.TitleSnapshot != "牙刷挖矿×3" {
		t.Fatalf("title=%s", o.TitleSnapshot)
	}
	list, err := ords.ListByUser(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("orders=%d", len(list))
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.RechargeBalance.Equal(decimal.RequireFromString("2000")) {
		t.Fatalf("recharge=%s", got.RechargeBalance)
	}
	if !got.CapEffective.Equal(decimal.RequireFromString("1800")) {
		t.Fatalf("cap=%s", got.CapEffective)
	}
}

func TestBuyCartWithRecharge_InsufficientFailsWholeCart(t *testing.T) {
	ResetRuntimeCapTiers()
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address: "0xabc", RechargeBalance: decimal.RequireFromString("2000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	ords := newMemOrders(users)
	uc := NewOrderUseCase(&memPackages{rows: []*Package{
		{ID: 1, Amount: decimal.RequireFromString("1000"), Title: "牙刷挖矿", Enabled: true},
	}}, ords, users, users, &memLedger{})
	if _, err := uc.BuyCartWithRecharge(context.Background(), u.ID, []CartItem{
		{GoodsID: 1, Qty: 3},
	}, 300); err != ErrInsufficientBalance {
		t.Fatalf("got %v", err)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.RechargeBalance.Equal(decimal.RequireFromString("2000")) {
		t.Fatalf("recharge=%s", got.RechargeBalance)
	}
	list, err := ords.ListByUser(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("orders=%d", len(list))
	}
}

func TestBuyWithRechargeGoods_SingleBuyUnchanged(t *testing.T) {
	ResetRuntimeCapTiers()
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address: "0xabc", RechargeBalance: decimal.RequireFromString("2000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewOrderUseCase(&memPackages{rows: []*Package{{
		ID: 9, Amount: decimal.RequireFromString("1000"), Title: "牙刷", Enabled: true,
	}}}, newMemOrders(users), users, users, &memLedger{})
	o, err := uc.BuyWithRechargeGoods(context.Background(), u.ID, 9, 300)
	if err != nil {
		t.Fatal(err)
	}
	if o.PackageID != 9 || !o.Amount.Equal(decimal.RequireFromString("1000")) {
		t.Fatalf("%+v", o)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.RechargeBalance.Equal(decimal.RequireFromString("1000")) {
		t.Fatalf("recharge=%s", got.RechargeBalance)
	}
	if !got.CapEffective.Equal(decimal.RequireFromString("600")) {
		t.Fatalf("cap=%s", got.CapEffective)
	}
}

func TestBuyCartWithRecharge_DisabledGoods(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address: "0xabc", RechargeBalance: decimal.RequireFromString("5000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewOrderUseCase(&memPackages{rows: []*Package{
		{ID: 1, Amount: decimal.RequireFromString("1000"), Title: "下架", Enabled: false},
	}}, newMemOrders(users), users, users, &memLedger{})
	if _, err := uc.BuyCartWithRecharge(context.Background(), u.ID, []CartItem{
		{GoodsID: 1, Qty: 1},
	}, 300); err != ErrPackageDisabled {
		t.Fatalf("got %v", err)
	}
}

func TestBuyWithRecharge_Insufficient(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xabc"})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewOrderUseCase(&memPackages{rows: []*Package{{
		ID: 1, Amount: decimal.RequireFromString("1000"), Title: "t", Enabled: true,
	}}}, newMemOrders(users), users, users, &memLedger{})
	if _, err := uc.BuyWithRecharge(context.Background(), u.ID, decimal.RequireFromString("1000"), 300); err != ErrInsufficientBalance {
		t.Fatalf("got %v", err)
	}
}

func TestCreateOrder_RequiresReleaseDays(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xabc"})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewOrderUseCase(&memPackages{rows: []*Package{{
		ID: 1, Amount: decimal.RequireFromString("1000"), Title: "t", Enabled: true,
	}}}, newMemOrders(users), users, users, &memLedger{})
	if _, err := uc.CreateOrder(context.Background(), u.ID, decimal.RequireFromString("1000"), 0); err != ErrInvalidReleaseDays {
		t.Fatalf("got %v", err)
	}
}

func TestUpdatePackage_AmountAndDays(t *testing.T) {
	pkgs := &memPackages{rows: []*Package{
		{ID: 1, Amount: decimal.RequireFromString("1000"), ReleaseDays: 300, Enabled: true},
		{ID: 2, Amount: decimal.RequireFromString("3000"), ReleaseDays: 300, Enabled: true},
	}}
	uc := NewOrderUseCase(pkgs, newMemOrders(newMemUsers()), newMemUsers(), newMemUsers(), &memLedger{})
	got, err := uc.UpdatePackage(context.Background(), 1, decimal.RequireFromString("1500"), 600)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Amount.Equal(decimal.RequireFromString("1500")) || got.ReleaseDays != 600 {
		t.Fatalf("%+v", got)
	}
	if _, err := uc.UpdatePackage(context.Background(), 1, decimal.RequireFromString("3000"), 300); err != ErrPackageAmountTaken {
		t.Fatalf("dup amount: %v", err)
	}
	if _, err := uc.UpdatePackage(context.Background(), 1, decimal.RequireFromString("1500"), 90); err != ErrInvalidReleaseDays {
		t.Fatalf("bad days: %v", err)
	}
}

func TestCreateSaveDeletePackage(t *testing.T) {
	pkgs := &memPackages{rows: []*Package{
		{ID: 1, Amount: decimal.RequireFromString("1000"), Title: "旧", ReleaseDays: 300, Enabled: true},
	}}
	uc := NewOrderUseCase(pkgs, newMemOrders(newMemUsers()), newMemUsers(), newMemUsers(), &memLedger{})
	got, err := uc.CreatePackage(context.Background(), &Package{
		Amount: decimal.RequireFromString("888"), Title: "新品", DailyCap: decimal.RequireFromString("400"),
		ReleaseDays: 600, SortOrder: 15, Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == 0 || got.GoodsDesc != "新品" || got.ReleaseDays != 600 {
		t.Fatalf("%+v", got)
	}
	if _, err := uc.CreatePackage(context.Background(), &Package{
		Amount: decimal.RequireFromString("1000"), Title: "重金额", ReleaseDays: 300, Enabled: true,
	}); err != ErrPackageAmountTaken {
		t.Fatalf("dup: %v", err)
	}
	got, err = uc.SavePackage(context.Background(), &Package{
		ID: got.ID, Amount: decimal.RequireFromString("888"), Title: "改名", GoodsDesc: "描述",
		DailyCap: decimal.RequireFromString("500"), ReleaseDays: 750, SortOrder: 20, Enabled: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "改名" || got.Enabled || got.ReleaseDays != 750 {
		t.Fatalf("save %+v", got)
	}
	if err := uc.DeletePackage(context.Background(), got.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.GetPackage(context.Background(), got.ID); err != ErrPackageNotFound {
		t.Fatalf("deleted: %v", err)
	}
}

func TestRandomOrderNo(t *testing.T) {
	a := RandomOrderNo()
	b := RandomOrderNo()
	if len(a) != 7 || a[0] != 'C' {
		t.Fatalf("a=%s", a)
	}
	for i := 1; i < 7; i++ {
		if a[i] < '0' || a[i] > '9' {
			t.Fatalf("a=%s", a)
		}
	}
	if a == b {
		t.Log("same twice is possible")
	}
}

func TestFormatOrderNoAndSource(t *testing.T) {
	if got := FormatOrderNo(16); got != "C000016" {
		t.Fatalf("no=%s", got)
	}
	if got := FormatOrderSource("C000016", "50000"); got != "C000016 / 50000" {
		t.Fatalf("source=%s", got)
	}
	o := &Order{ID: 8}
	if o.DisplayNo() != "C000008" {
		t.Fatalf("display=%s", o.DisplayNo())
	}
	o.OrderNo = "C000099"
	if o.DisplayNo() != "C000099" {
		t.Fatalf("override=%s", o.DisplayNo())
	}
}
