package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

func TestAdjust_AddAndSubAvailableAndLock(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
	if err != nil {
		t.Fatal(err)
	}
	led := &memLedger{}
	uc := NewAdjustUseCase(users, users, led, NopTx{})

	got, err := uc.Adjust(context.Background(), u.Address, AdjustAvailable, decimal.RequireFromString("10.5"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.AvailableBalance.Equal(decimal.RequireFromString("10.5")) {
		t.Fatalf("avail=%s", got.AvailableBalance)
	}
	got, err = uc.Adjust(context.Background(), u.Address, AdjustLock, decimal.RequireFromString("3"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockBalance.Equal(decimal.RequireFromString("3")) {
		t.Fatalf("lock=%s", got.LockBalance)
	}
	got, err = uc.Adjust(context.Background(), u.Address, AdjustAvailable, decimal.RequireFromString("-4"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.AvailableBalance.Equal(decimal.RequireFromString("6.5")) || !got.LockBalance.Equal(decimal.RequireFromString("3")) {
		t.Fatalf("avail=%s lock=%s", got.AvailableBalance, got.LockBalance)
	}
	if len(led.rows) != 3 {
		t.Fatalf("ledger=%d", len(led.rows))
	}
	if led.rows[0].EntryType != LedgerAdminAdjust || led.rows[0].BalanceKind != AdjustAvailable {
		t.Fatalf("%+v", led.rows[0])
	}
}

func TestAdjust_IspayAndLockIspay(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewAdjustUseCase(users, users, &memLedger{}, NopTx{})
	got, err := uc.Adjust(context.Background(), u.Address, AdjustIspay, decimal.RequireFromString("1.2"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.IspayBalance.Equal(decimal.RequireFromString("1.2")) {
		t.Fatalf("ispay=%s", got.IspayBalance)
	}
	got, err = uc.Adjust(context.Background(), u.Address, AdjustLockIspay, decimal.RequireFromString("0.5"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Adjust(context.Background(), u.Address, AdjustLockIspay, decimal.RequireFromString("-0.2")); err != nil {
		t.Fatal(err)
	}
	got, err = users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LockIspay.Equal(decimal.RequireFromString("0.3")) {
		t.Fatalf("lock ispay=%s", got.LockIspay)
	}
}

func TestAdjust_RejectsZeroKindAndOverdraft(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{Address: "0xcccccccccccccccccccccccccccccccccccccccc"})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewAdjustUseCase(users, users, &memLedger{}, NopTx{})
	if _, err := uc.Adjust(context.Background(), u.Address, AdjustAvailable, decimal.Zero); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("zero: %v", err)
	}
	got, err := uc.Adjust(context.Background(), u.Address, AdjustRecharge, decimal.RequireFromString("20"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.RechargeBalance.Equal(decimal.RequireFromString("20")) {
		t.Fatalf("recharge=%s", got.RechargeBalance)
	}
	if _, err := uc.Adjust(context.Background(), u.Address, "frozen", decimal.RequireFromString("1")); !errors.Is(err, ErrAdjustKind) {
		t.Fatalf("kind: %v", err)
	}
	if _, err := uc.Adjust(context.Background(), u.Address, AdjustLock, decimal.RequireFromString("-1")); !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("overdraft: %v", err)
	}
	if _, err := uc.Adjust(context.Background(), "not-an-address", AdjustAvailable, decimal.RequireFromString("1")); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("addr: %v", err)
	}
}
