package service

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestReadWeb3GoodsReqJSONContentsAndSKUs(t *testing.T) {
	body := `{
		"id":"12",
		"name":"牙刷",
		"desc":"中文描述",
		"amount":"1000",
		"on_sale":1,
		"contents":{
			"zh":{"title":"牙刷","desc":"中文描述","image":"/uploads/zh.png","detail":"<p>中文</p>"},
			"en":{"title":"Brush","desc":"English desc","image":"/uploads/en.png","detail":"<p>EN</p>"}
		},
		"skus":[{"id":9,"name":"白色","name_en":"White","amount":"800","image":"/uploads/sku.png","enabled":1}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	in, err := readWeb3GoodsReq(req)
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	assertWeb3GoodsNested(t, in)
}

func TestReadWeb3GoodsReqFormBracketContentsAndSKUs(t *testing.T) {
	form := url.Values{}
	form.Set("id", "12")
	form.Set("name", "牙刷")
	form.Set("desc", "中文描述")
	form.Set("amount", "1000")
	form.Set("on_sale", "1")
	form.Set("contents[zh][title]", "牙刷")
	form.Set("contents[zh][desc]", "中文描述")
	form.Set("contents[zh][image]", "/uploads/zh.png")
	form.Set("contents[zh][detail]", "<p>中文</p>")
	form.Set("contents[en][title]", "Brush")
	form.Set("contents[en][desc]", "English desc")
	form.Set("contents[en][image]", "/uploads/en.png")
	form.Set("contents[en][detail]", "<p>EN</p>")
	form.Set("skus[0][id]", "9")
	form.Set("skus[0][name]", "白色")
	form.Set("skus[0][name_en]", "White")
	form.Set("skus[0][amount]", "800")
	form.Set("skus[0][image]", "/uploads/sku.png")
	form.Set("skus[0][enabled]", "1")
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	in, err := readWeb3GoodsReq(req)
	if err != nil {
		t.Fatalf("form brackets: %v", err)
	}
	assertWeb3GoodsNested(t, in)
}

func TestReadWeb3GoodsReqFormJSONContentsAndSKUs(t *testing.T) {
	form := url.Values{}
	form.Set("id", "12")
	form.Set("name", "牙刷")
	form.Set("desc", "中文描述")
	form.Set("amount", "1000")
	form.Set("on_sale", "1")
	form.Set("contents", `{"zh":{"title":"牙刷","desc":"中文描述","image":"/uploads/zh.png","detail":"<p>中文</p>"},"en":{"title":"Brush","desc":"English desc","image":"/uploads/en.png","detail":"<p>EN</p>"}}`)
	form.Set("skus", `[{"id":9,"name":"白色","name_en":"White","amount":"800","image":"/uploads/sku.png","enabled":1}]`)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	in, err := readWeb3GoodsReq(req)
	if err != nil {
		t.Fatalf("form json: %v", err)
	}
	assertWeb3GoodsNested(t, in)
}

func TestReadWeb3GoodsReqFormEmptySKUArray(t *testing.T) {
	form := url.Values{}
	form.Set("id", "12")
	form.Set("name", "牙刷")
	form.Set("amount", "1000")
	form.Set("skus", "[]")
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	in, err := readWeb3GoodsReq(req)
	if err != nil {
		t.Fatalf("empty skus: %v", err)
	}
	if !in.HasSKUs || len(in.SKUs) != 0 {
		t.Fatalf("want cleared skus, got has=%v skus=%+v", in.HasSKUs, in.SKUs)
	}
}

func assertWeb3GoodsNested(t *testing.T, in *web3GoodsReq) {
	t.Helper()
	if in == nil || in.ID != 12 {
		t.Fatalf("id: %+v", in)
	}
	en := in.Contents["en"]
	if strings.TrimSpace(en.Title) != "Brush" || strings.TrimSpace(en.GoodsDesc) != "English desc" ||
		en.Image != "/uploads/en.png" || en.Detail != "<p>EN</p>" {
		t.Fatalf("en content %+v", en)
	}
	zh := in.Contents["zh"]
	if zh.Title != "牙刷" || in.Name != "牙刷" {
		t.Fatalf("zh content %+v name=%s", zh, in.Name)
	}
	if !in.HasSKUs || len(in.SKUs) != 1 {
		t.Fatalf("skus %+v has=%v", in.SKUs, in.HasSKUs)
	}
	sku := in.SKUs[0]
	if sku.ID != 9 || sku.Name != "白色" || sku.NameEn != "White" || sku.Amount.String() != "800" || !sku.Enabled {
		t.Fatalf("sku %+v", sku)
	}
}
