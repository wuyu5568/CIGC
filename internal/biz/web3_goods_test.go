package biz

import (
	"context"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func TestListWeb3GoodsPaginatesWithoutDays(t *testing.T) {
	pkgs := &memPackages{rows: []*Package{
		{ID: 1, Amount: decimal.RequireFromString("1000"), Title: "a", ReleaseDays: 300, Enabled: true},
		{ID: 2, Amount: decimal.RequireFromString("2000"), Title: "b", ReleaseDays: 300, Enabled: false},
		{ID: 3, Amount: decimal.RequireFromString("3000"), Title: "c", ReleaseDays: 600, Enabled: true},
		{ID: 4, Amount: decimal.RequireFromString("4000"), Title: "d", ReleaseDays: 750, Enabled: true},
	}}
	uc := NewOrderUseCase(pkgs, newMemOrders(newMemUsers()), newMemUsers(), newMemUsers(), &memLedger{})
	if _, _, err := uc.ListWeb3Goods(context.Background(), 90, 1, 10, false); err != ErrInvalidReleaseDays {
		t.Fatalf("bad days: %v", err)
	}
	all, total, err := uc.ListWeb3Goods(context.Background(), 0, 1, 10, false)
	if err != nil || total != 4 || len(all) != 4 {
		t.Fatalf("all: total=%d rows=%+v err=%v", total, all, err)
	}
	onSale, total, err := uc.ListWeb3Goods(context.Background(), 0, 1, 10, true)
	if err != nil || total != 3 || len(onSale) != 3 {
		t.Fatalf("user on-sale: total=%d rows=%+v err=%v", total, onSale, err)
	}
	page, total, err := uc.ListWeb3Goods(context.Background(), 0, 2, 1, false)
	if err != nil || total != 4 || len(page) != 1 || page[0].ID != 2 {
		t.Fatalf("page2: total=%d page=%+v err=%v", total, page, err)
	}
	six, total, err := uc.ListWeb3Goods(context.Background(), 600, 1, 10, false)
	if err != nil || total != 1 || len(six) != 1 || six[0].ID != 3 {
		t.Fatalf("600 filter: total=%d %+v err=%v", total, six, err)
	}
}

func TestCreateWeb3GoodsRequiresNameDesc(t *testing.T) {
	pkgs := &memPackages{rows: []*Package{
		{ID: 1, Amount: decimal.RequireFromString("1000"), Title: "旧", GoodsDesc: "d", ReleaseDays: 300, Enabled: true},
	}}
	uc := NewOrderUseCase(pkgs, newMemOrders(newMemUsers()), newMemUsers(), newMemUsers(), &memLedger{})
	if _, err := uc.CreateWeb3Goods(context.Background(), &Web3GoodsInput{
		Desc: "y", Amount: decimal.RequireFromString("2000"),
	}); err != ErrPackageTitle {
		t.Fatalf("name: %v", err)
	}
	if _, err := uc.CreateWeb3Goods(context.Background(), &Web3GoodsInput{
		Name: "x", Amount: decimal.RequireFromString("2000"),
	}); err != ErrPackageDesc {
		t.Fatalf("desc: %v", err)
	}
	got, err := uc.CreateWeb3Goods(context.Background(), &Web3GoodsInput{
		Name: "牙刷", Desc: "描述", Amount: decimal.RequireFromString("2000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "牙刷" || got.ReleaseDays != ReleaseDays300 || !got.Enabled || !got.DailyCap.IsZero() {
		t.Fatalf("%+v", got)
	}
	sameAmt, err := uc.CreateWeb3Goods(context.Background(), &Web3GoodsInput{
		Name: "同价", Desc: "描述2", Amount: decimal.RequireFromString("2000"),
	})
	if err != nil || sameAmt.ID == got.ID {
		t.Fatalf("same amount allowed: %+v err=%v", sameAmt, err)
	}
}

func TestUpdateWeb3GoodsByID(t *testing.T) {
	pkgs := &memPackages{rows: []*Package{
		{ID: 1, Amount: decimal.RequireFromString("1000"), Title: "a", GoodsDesc: "da", ReleaseDays: 300, Enabled: true, Image: "/uploads/old.png"},
		{ID: 2, Amount: decimal.RequireFromString("2000"), Title: "b", GoodsDesc: "db", ReleaseDays: 600, Enabled: true},
	}}
	uc := NewOrderUseCase(pkgs, newMemOrders(newMemUsers()), newMemUsers(), newMemUsers(), &memLedger{})
	got, err := uc.UpdateWeb3Goods(context.Background(), &Web3GoodsInput{
		ID: 1, Name: "改名", Desc: "新描述", Amount: decimal.RequireFromString("1500"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "改名" || got.Image != "/uploads/old.png" || got.ReleaseDays != 300 {
		t.Fatalf("keep image %+v", got)
	}
	got, err = uc.UpdateWeb3Goods(context.Background(), &Web3GoodsInput{
		ID: 1, Name: "改名", Desc: "新描述", Amount: decimal.RequireFromString("1500"),
		HasImage: true, Image: "/uploads/new.png",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Image != "/uploads/new.png" {
		t.Fatalf("replace image %+v", got)
	}
	off, err := uc.SetWeb3GoodsOnSale(context.Background(), 2, false)
	if err != nil || off.Enabled {
		t.Fatalf("status %+v err=%v", off, err)
	}
	if err := uc.DeleteWeb3Goods(context.Background(), 2); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestGetWeb3GoodsOnSaleOnly(t *testing.T) {
	pkgs := &memPackages{rows: []*Package{
		{ID: 1, Amount: decimal.RequireFromString("1000"), Title: "a", GoodsDesc: "da", Enabled: true},
		{ID: 2, Amount: decimal.RequireFromString("2000"), Title: "b", GoodsDesc: "db", Enabled: false},
	}}
	uc := NewOrderUseCase(pkgs, newMemOrders(newMemUsers()), newMemUsers(), newMemUsers(), &memLedger{})
	got, err := uc.GetWeb3Goods(context.Background(), 1, true)
	if err != nil || got.ID != 1 {
		t.Fatalf("on sale %+v err=%v", got, err)
	}
	if _, err := uc.GetWeb3Goods(context.Background(), 2, true); err != ErrPackageNotFound {
		t.Fatalf("off sale: %v", err)
	}
	got, err = uc.GetWeb3Goods(context.Background(), 2, false)
	if err != nil || got.ID != 2 {
		t.Fatalf("admin off sale %+v err=%v", got, err)
	}
}

func TestWeb3GoodsDetailCreateAndKeepOnUpdate(t *testing.T) {
	pkgs := &memPackages{}
	uc := NewOrderUseCase(pkgs, newMemOrders(newMemUsers()), newMemUsers(), newMemUsers(), &memLedger{})
	got, err := uc.CreateWeb3Goods(context.Background(), &Web3GoodsInput{
		Name: "牙刷", Desc: "描述", Amount: decimal.RequireFromString("2000"),
		HasDetail: true, Detail: `<p onclick="x">商品详情</p><script>alert(1)</script>`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Detail, "商品详情") || strings.Contains(strings.ToLower(got.Detail), "script") || strings.Contains(strings.ToLower(got.Detail), "onclick") {
		t.Fatalf("create detail %+v", got)
	}
	kept := got.Detail
	got, err = uc.UpdateWeb3Goods(context.Background(), &Web3GoodsInput{
		ID: got.ID, Name: "牙刷", Desc: "描述", Amount: decimal.RequireFromString("2000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Detail != kept {
		t.Fatalf("keep detail %+v", got)
	}
	got, err = uc.UpdateWeb3Goods(context.Background(), &Web3GoodsInput{
		ID: got.ID, Name: "牙刷", Desc: "描述", Amount: decimal.RequireFromString("2000"),
		HasDetail: true, Detail: "<p>新详情</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Detail, "新详情") {
		t.Fatalf("replace detail %+v", got)
	}
}
