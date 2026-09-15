package service

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/cigc/app/internal/biz"
)

func TestParseBuyCartItems(t *testing.T) {
	cart, has, err := parseBuyCartItems(nil)
	if err != nil || has || cart != nil {
		t.Fatalf("empty raw: cart=%v has=%v err=%v", cart, has, err)
	}
	cart, has, err = parseBuyCartItems(json.RawMessage(`[]`))
	if err != nil || has {
		t.Fatalf("empty array: has=%v err=%v", has, err)
	}
	cart, has, err = parseBuyCartItems(json.RawMessage(`[{"id":1,"qty":2},{"id":"3","qty":"1"}]`))
	if err != nil || !has {
		t.Fatalf("ok: has=%v err=%v", has, err)
	}
	if len(cart) != 2 || cart[0] != (biz.CartItem{GoodsID: 1, Qty: 2}) || cart[1] != (biz.CartItem{GoodsID: 3, Qty: 1}) {
		t.Fatalf("cart=%+v", cart)
	}
	if _, _, err := parseBuyCartItems(json.RawMessage(`[{"id":1,"qty":0}]`)); !errors.Is(err, biz.ErrInvalidAmount) {
		t.Fatalf("qty0: %v", err)
	}
	if _, _, err := parseBuyCartItems(json.RawMessage(`[{"id":0,"qty":1}]`)); !errors.Is(err, biz.ErrPackageNotFound) {
		t.Fatalf("id0: %v", err)
	}
}
