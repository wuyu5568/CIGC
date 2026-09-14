package biz

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
)

func TestListWeb3GoodsRequiresDaysAndPaginates(t *testing.T) {
	pkgs := &memPackages{rows: []*Package{
		{ID: 1, Amount: decimal.RequireFromString("1000"), Title: "a", ReleaseDays: 300, Enabled: true},
		{ID: 2, Amount: decimal.RequireFromString("2000"), Title: "b", ReleaseDays: 300, Enabled: false},
		{ID: 3, Amount: decimal.RequireFromString("3000"), Title: "c", ReleaseDays: 600, Enabled: true},
		{ID: 4, Amount: decimal.RequireFromString("4000"), Title: "d", ReleaseDays: 750, Enabled: true},
	}}
	uc := NewOrderUseCase(pkgs, newMemOrders(newMemUsers()), newMemUsers(), newMemUsers(), &memLedger{})
	if _, _, err := uc.ListWeb3Goods(context.Background(), 0, 1, 10, false); err != ErrInvalidReleaseDays {
		t.Fatalf("missing days: %v", err)
	}
	if _, _, err := uc.ListWeb3Goods(context.Background(), 90, 1, 10, false); err != ErrInvalidReleaseDays {
		t.Fatalf("bad days: %v", err)
	}
	rows, total, err := uc.ListWeb3Goods(context.Background(), 300, 1, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(rows) != 2 || rows[0].ID != 1 || rows[1].ID != 2 {
		t.Fatalf("300 list total=%d rows=%+v", total, rows)
	}
	onSale, total, err := uc.ListWeb3Goods(context.Background(), 300, 1, 10, true)
	if err != nil || total != 1 || len(onSale) != 1 || onSale[0].ID != 1 {
		t.Fatalf("user on-sale: total=%d rows=%+v err=%v", total, onSale, err)
	}
	page, total, err := uc.ListWeb3Goods(context.Background(), 300, 2, 1, false)
	if err != nil || total != 2 || len(page) != 1 || page[0].ID != 2 {
		t.Fatalf("page2: total=%d page=%+v err=%v", total, page, err)
	}
	six, total, err := uc.ListWeb3Goods(context.Background(), 600, 1, 10, false)
	if err != nil || total != 1 || len(six) != 1 || six[0].ID != 3 {
		t.Fatalf("600: total=%d %+v err=%v", total, six, err)
	}
}

func TestCreateWeb3GoodsLockedToDays(t *testing.T) {
	pkgs := &memPackages{rows: []*Package{
		{ID: 1, Amount: decimal.RequireFromString("1000"), Title: "旧", GoodsDesc: "d", ReleaseDays: 300, Enabled: true},
	}}
	uc := NewOrderUseCase(pkgs, newMemOrders(newMemUsers()), newMemUsers(), newMemUsers(), &memLedger{})
	if _, err := uc.CreateWeb3Goods(context.Background(), &Web3GoodsInput{
		Name: "x", Desc: "y", Amount: decimal.RequireFromString("2000"), Days: 90,
	}); err != ErrInvalidReleaseDays {
		t.Fatalf("bad days: %v", err)
	}
	if _, err := uc.CreateWeb3Goods(context.Background(), &Web3GoodsInput{
		Amount: decimal.RequireFromString("2000"), Days: 600,
	}); err != ErrPackageDesc {
		t.Fatalf("desc: %v", err)
	}
	got, err := uc.CreateWeb3Goods(context.Background(), &Web3GoodsInput{
		Desc: "描述", Amount: decimal.RequireFromString("2000"), Days: 600,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "描述" || got.ReleaseDays != 600 || !got.Enabled || !got.DailyCap.IsZero() {
		t.Fatalf("%+v", got)
	}
	if _, err := uc.CreateWeb3Goods(context.Background(), &Web3GoodsInput{
		Name: "重复", Desc: "描述", Amount: decimal.RequireFromString("2000"), Days: 600,
	}); err != ErrPackageAmountTaken {
		t.Fatalf("dup same days: %v", err)
	}
	sameAmt, err := uc.CreateWeb3Goods(context.Background(), &Web3GoodsInput{
		Desc: "300同金额", Amount: decimal.RequireFromString("1000"), Days: 600,
	})
	if err != nil || sameAmt.ReleaseDays != 600 || !sameAmt.Amount.Equal(decimal.RequireFromString("1000")) {
		t.Fatalf("same amount other days: %+v err=%v", sameAmt, err)
	}
	list, total, err := uc.ListWeb3Goods(context.Background(), 300, 1, 10, false)
	if err != nil || total != 1 || list[0].ID != 1 {
		t.Fatalf("300 should not see 600 goods: total=%d %+v err=%v", total, list, err)
	}
}

func TestUpdateWeb3GoodsNoCrossCategory(t *testing.T) {
	pkgs := &memPackages{rows: []*Package{
		{ID: 1, Amount: decimal.RequireFromString("1000"), Title: "a", GoodsDesc: "da", ReleaseDays: 300, Enabled: true, Image: "/uploads/old.png"},
		{ID: 2, Amount: decimal.RequireFromString("2000"), Title: "b", GoodsDesc: "db", ReleaseDays: 600, Enabled: true},
	}}
	uc := NewOrderUseCase(pkgs, newMemOrders(newMemUsers()), newMemUsers(), newMemUsers(), &memLedger{})
	if _, err := uc.UpdateWeb3Goods(context.Background(), &Web3GoodsInput{
		ID: 2, Name: "改", Desc: "描述", Amount: decimal.RequireFromString("2000"), Days: 300,
	}); err != ErrPackageNotFound {
		t.Fatalf("cross update: %v", err)
	}
	got, err := uc.UpdateWeb3Goods(context.Background(), &Web3GoodsInput{
		ID: 1, Name: "改名", Desc: "新描述", Amount: decimal.RequireFromString("1500"), Days: 300,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "改名" || got.Image != "/uploads/old.png" || got.ReleaseDays != 300 {
		t.Fatalf("keep image %+v", got)
	}
	got, err = uc.UpdateWeb3Goods(context.Background(), &Web3GoodsInput{
		ID: 1, Name: "改名", Desc: "新描述", Amount: decimal.RequireFromString("1500"), Days: 300,
		HasImage: true, Image: "/uploads/new.png",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Image != "/uploads/new.png" {
		t.Fatalf("replace image %+v", got)
	}
	if _, err := uc.SetWeb3GoodsOnSale(context.Background(), 2, 300, false); err != ErrPackageNotFound {
		t.Fatalf("cross status: %v", err)
	}
	if err := uc.DeleteWeb3Goods(context.Background(), 2, 300); err != ErrPackageNotFound {
		t.Fatalf("cross delete: %v", err)
	}
	off, err := uc.SetWeb3GoodsOnSale(context.Background(), 1, 300, false)
	if err != nil || off.Enabled {
		t.Fatalf("status %+v err=%v", off, err)
	}
}
