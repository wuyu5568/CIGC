package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/cigc/app/internal/conf"
	"github.com/shopspring/decimal"
)

func TestListAdminOrders(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xabc"})
	if err != nil {
		t.Fatal(err)
	}
	orders := newMemOrders(users)
	uc := NewOrderUseCase(&memPackages{rows: []*Package{{
		ID: 1, Amount: decimal.RequireFromString("1000"), Title: "牙刷", Enabled: true,
	}}}, orders, users)
	o1, err := uc.CreateOrder(context.Background(), u.ID, decimal.RequireFromString("1000"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.MarkPaid(context.Background(), o1.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.CreateOrder(context.Background(), u.ID, decimal.RequireFromString("1000")); err != nil {
		t.Fatal(err)
	}

	all, err := uc.ListAdminOrders(context.Background(), "", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if all.Total != 2 {
		t.Fatalf("total=%d", all.Total)
	}
	pending, err := uc.ListAdminOrders(context.Background(), "", OrderPending, 1)
	if err != nil {
		t.Fatal(err)
	}
	if pending.Total != 1 || pending.Items[0].Status != OrderPending {
		t.Fatalf("%+v", pending)
	}
	byAddr, err := uc.ListAdminOrders(context.Background(), "0xabc", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if byAddr.Total != 2 || byAddr.Items[0].Address != "0xabc" {
		t.Fatalf("%+v", byAddr)
	}
}

func TestListAdminUsers_AndLock(t *testing.T) {
	users := newMemUsers()
	uc := NewUserUseCase(users, &memRecommends{path: map[uint64]string{}}, stubVerifier{}, stubTokens{}, &conf.Auth{}, addrGenesis, nil)
	if _, err := uc.EthAuthorize(context.Background(), addrGenesis, "sig", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.EthAuthorize(context.Background(), addrUserB, "sig", addrGenesis); err != nil {
		t.Fatal(err)
	}

	page, err := uc.ListAdminUsers(context.Background(), "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 {
		t.Fatalf("total=%d", page.Total)
	}

	b, err := users.FindByAddress(context.Background(), addrUserB)
	if err != nil {
		t.Fatal(err)
	}
	if err := uc.SetUserLock(context.Background(), b.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := uc.SetUserLock(context.Background(), b.ID, true); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(context.Background(), b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsDisabled() {
		t.Fatal("expected locked")
	}
	if _, err := uc.EthAuthorize(context.Background(), addrUserB, "sig", ""); !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("login after lock: %v", err)
	}
	if err := uc.UnlockUser(context.Background(), b.ID); err != nil {
		t.Fatal(err)
	}
	got, err = users.FindByID(context.Background(), b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.IsDisabled() {
		t.Fatal("expected unlocked")
	}
}

func TestUnsupportedSubMoneyError(t *testing.T) {
	if ErrUnsupportedOperation.Reason != "UNSUPPORTED_OPERATION" {
		t.Fatalf("reason=%s", ErrUnsupportedOperation.Reason)
	}
}
