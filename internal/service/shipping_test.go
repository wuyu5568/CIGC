package service

import (
	"testing"

	"github.com/cigc/app/internal/biz"
)

func TestShippingJSON(t *testing.T) {
	empty := shippingJSON(nil)
	if empty["name"] != "" || empty["contact"] != "" || empty["address"] != "" {
		t.Fatalf("empty: %+v", empty)
	}
	got := shippingJSON(&biz.ShippingAddress{Name: "张三", Contact: "138", Address: "北京"})
	if got["name"] != "张三" || got["contact"] != "138" || got["address"] != "北京" {
		t.Fatalf("got %+v", got)
	}
}
