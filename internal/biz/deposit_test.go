package biz

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/cigc/app/internal/conf"
	"github.com/shopspring/decimal"
)

type memDeposits struct {
	byKey map[string]*ChainDeposit
}

func newMemDeposits() *memDeposits {
	return &memDeposits{byKey: map[string]*ChainDeposit{}}
}

func depositKey(tx string, idx int) string {
	return tx + "#" + strconv.Itoa(idx)
}

func (m *memDeposits) FindByEvent(_ context.Context, txHash string, logIndex int) (*ChainDeposit, error) {
	d, ok := m.byKey[depositKey(txHash, logIndex)]
	if !ok {
		return nil, nil
	}
	cp := *d
	return &cp, nil
}

func (m *memDeposits) ListByOrder(_ context.Context, orderID uint64) ([]*ChainDeposit, error) {
	var out []*ChainDeposit
	for _, d := range m.byKey {
		if d.OrderID != nil && *d.OrderID == orderID && d.Status == DepositMatched {
			cp := *d
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (m *memDeposits) ListByFrom(_ context.Context, fromAddr string, page, pageSize int) ([]*ChainDeposit, int, error) {
	return m.listFiltered(func(d *ChainDeposit) bool { return d.FromAddr == fromAddr }, page, pageSize)
}

func (m *memDeposits) ListAdmin(_ context.Context, fromAddr string, page, pageSize int) ([]*ChainDeposit, int, error) {
	return m.listFiltered(func(d *ChainDeposit) bool {
		return fromAddr == "" || containsFold(d.FromAddr, fromAddr)
	}, page, pageSize)
}

func (m *memDeposits) listFiltered(keep func(*ChainDeposit) bool, page, pageSize int) ([]*ChainDeposit, int, error) {
	var all []*ChainDeposit
	for _, d := range m.byKey {
		if keep(d) {
			cp := *d
			all = append(all, &cp)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID > all[j].ID })
	total := len(all)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultRewardPageSize
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []*ChainDeposit{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return all[start:end], total, nil
}

func (m *memDeposits) Create(_ context.Context, d *ChainDeposit) error {
	k := depositKey(d.TxHash, d.LogIndex)
	if _, ok := m.byKey[k]; ok {
		return ErrOrderConflict
	}
	cp := *d
	cp.ID = uint64(len(m.byKey) + 1)
	m.byKey[k] = &cp
	return nil
}

type memCursors struct{ n map[string]uint64 }

func newMemCursors() *memCursors { return &memCursors{n: map[string]uint64{}} }

func (m *memCursors) Get(_ context.Context, name string) (uint64, error) { return m.n[name], nil }
func (m *memCursors) Set(_ context.Context, name string, block uint64) error {
	m.n[name] = block
	return nil
}

type memChain struct {
	head uint64
	logs []*ChainTransfer
}

func (m *memChain) BlockNumber(_ context.Context) (uint64, error) { return m.head, nil }
func (m *memChain) ListTransfers(_ context.Context, _, recv string, from, to uint64) ([]*ChainTransfer, error) {
	recv = strings.ToLower(recv)
	var out []*ChainTransfer
	for _, tr := range m.logs {
		if tr.BlockNumber >= from && tr.BlockNumber <= to {
			if recv != "" && strings.ToLower(tr.To) != recv {
				continue
			}
			cp := *tr
			out = append(out, &cp)
		}
	}
	return out, nil
}

func TestRawTokenToDecimal(t *testing.T) {
	// 1000 USDT with 18 decimals
	raw := "0x" + "00000000000000000000000000000000000000000000003635c9adc5dea00000"
	got, err := rawTokenToDecimal(raw, 18)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(decimal.RequireFromString("1000")) {
		t.Fatalf("got %s", got)
	}
	got, err = rawTokenToDecimal("1000000000000000000000", 18)
	if err != nil || !got.Equal(decimal.RequireFromString("1000")) {
		t.Fatalf("%s %v", got, err)
	}
}

func TestDeposit_CreditsRechargeAndIdempotent(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	if _, err := orders.Create(context.Background(), &Order{
		UserID: u.ID, PackageID: 1, Amount: decimal.RequireFromString("1000"), Status: OrderPending,
	}); err != nil {
		t.Fatal(err)
	}
	led := &memLedger{}
	uc := NewDepositUseCase(users, orders, newMemDeposits(), newMemCursors(), &memChain{head: 100}, NopTx{}, &conf.App{
		ReceiveAddress:       "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		UsdtAddress:          "0x55d398326f99059ff775485246999027b3197955",
		DepositConfirmations: 1,
	}, led)
	st, err := uc.ProcessTransfer(context.Background(), &ChainTransfer{
		TxHash: "0xabc1", LogIndex: 0,
		From: u.Address, To: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		AmountRaw: "1000000000000000000000", BlockNumber: 90,
	})
	if err != nil || st != DepositMatched {
		t.Fatalf("st=%s err=%v", st, err)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil || !got.RechargeBalance.Equal(decimal.RequireFromString("1000")) {
		t.Fatalf("recharge=%v err=%v", got, err)
	}
	st2, err := uc.ProcessTransfer(context.Background(), &ChainTransfer{
		TxHash: "0xabc1", LogIndex: 0,
		From: u.Address, To: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		AmountRaw: "1000000000000000000000", BlockNumber: 90,
	})
	if err != nil || st2 != DepositSkipped {
		t.Fatalf("dup st=%s err=%v", st2, err)
	}
	got, _ = users.FindByID(context.Background(), u.ID)
	if !got.RechargeBalance.Equal(decimal.RequireFromString("1000")) {
		t.Fatalf("dup credited again: %s", got.RechargeBalance)
	}
}

func TestDeposit_AbnormalUnknownSender(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	uc := NewDepositUseCase(users, orders, newMemDeposits(), newMemCursors(), &memChain{head: 100}, NopTx{}, &conf.App{
		ReceiveAddress: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		UsdtAddress:    "0x55d398326f99059ff775485246999027b3197955",
	}, &memLedger{})
	st, err := uc.ProcessTransfer(context.Background(), &ChainTransfer{
		TxHash: "0xbad1", LogIndex: 0,
		From:      "0xcccccccccccccccccccccccccccccccccccccccc",
		To:        "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		AmountRaw: "1000000000000000000000",
	})
	if err != nil || st != DepositAbnormal {
		t.Fatalf("unknown: %s %v", st, err)
	}
	got, _ := users.FindByID(context.Background(), u.ID)
	if !got.RechargeBalance.IsZero() {
		t.Fatalf("unknown sender credited: %s", got.RechargeBalance)
	}
}

func TestDeposit_ScanUsesCursor(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	if _, err := orders.Create(context.Background(), &Order{
		UserID: u.ID, PackageID: 1, Amount: decimal.RequireFromString("1000"), Status: OrderPending,
	}); err != nil {
		t.Fatal(err)
	}
	recv := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	chain := &memChain{
		head: 120,
		logs: []*ChainTransfer{{
			TxHash: "0xscan1", LogIndex: 1, From: u.Address, To: recv,
			AmountRaw: "1000000000000000000000", BlockNumber: 100,
		}},
	}
	cur := newMemCursors()
	_ = cur.Set(context.Background(), ChainCursorDeposit, 90)
	uc := NewDepositUseCase(users, orders, newMemDeposits(), cur, chain, NopTx{}, &conf.App{
		ReceiveAddress:       recv,
		UsdtAddress:          "0x55d398326f99059ff775485246999027b3197955",
		DepositConfirmations: 12,
	}, &memLedger{})
	res, err := uc.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Matched != 1 || res.FromBlock != 91 || res.ToBlock != 108 {
		t.Fatalf("%+v", res)
	}
	n, _ := cur.Get(context.Background(), ChainCursorDeposit)
	if n != 108 {
		t.Fatalf("cursor=%d", n)
	}
}

func TestDeposit_SplitSharesCreditSum(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	a := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	b := "0xcccccccccccccccccccccccccccccccccccccccc"
	uc := NewDepositUseCase(users, orders, newMemDeposits(), newMemCursors(), &memChain{head: 100}, NopTx{}, &conf.App{
		UsdtAddress: "0x55d398326f99059ff775485246999027b3197955",
		ReceiveAddresses: []conf.ReceiveShare{
			{Address: a, Percent: "75"},
			{Address: b, Percent: "25"},
		},
	}, &memLedger{})
	st, err := uc.ProcessTransfer(context.Background(), &ChainTransfer{
		TxHash: "0xsp1", LogIndex: 0, From: u.Address, To: a,
		AmountRaw: "750000000000000000000",
	})
	if err != nil || st != DepositMatched {
		t.Fatalf("share1: %s %v", st, err)
	}
	st, err = uc.ProcessTransfer(context.Background(), &ChainTransfer{
		TxHash: "0xsp2", LogIndex: 0, From: u.Address, To: b,
		AmountRaw: "250000000000000000000",
	})
	if err != nil || st != DepositMatched {
		t.Fatalf("share2: %s %v", st, err)
	}
	got, _ := users.FindByID(context.Background(), u.ID)
	if !got.RechargeBalance.Equal(decimal.RequireFromString("1000")) {
		t.Fatalf("recharge=%s", got.RechargeBalance)
	}
}

func TestDeposit_KeepsWrittenReceiveAddress(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	if _, err := orders.Create(context.Background(), &Order{
		UserID: u.ID, PackageID: 1, Amount: decimal.RequireFromString("1000"), Status: OrderPending,
	}); err != nil {
		t.Fatal(err)
	}
	written := "0x573ffd2739a3be7054ea6c3bb3581349f0745e8"
	uc := NewDepositUseCase(users, orders, newMemDeposits(), newMemCursors(), &memChain{head: 100}, NopTx{}, &conf.App{
		UsdtAddress: "0x55d398326f99059ff775485246999027b3197955",
		ReceiveAddresses: []conf.ReceiveShare{
			{Address: written, Percent: "100"},
		},
	}, &memLedger{})
	if got := uc.Shares()[0].Address; got != written {
		t.Fatalf("share address=%s", got)
	}
	st, err := uc.ProcessTransfer(context.Background(), &ChainTransfer{
		TxHash: "0xkeep1", LogIndex: 0, From: u.Address,
		To:        "0x0573ffd2739a3be7054ea6c3bb3581349f0745e8",
		AmountRaw: "1000000000000000000000",
	})
	if err != nil || st != DepositMatched {
		t.Fatalf("st=%s err=%v", st, err)
	}
}
