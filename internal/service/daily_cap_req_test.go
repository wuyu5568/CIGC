package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/cigc/app/internal/biz"
)

func TestReadDailyCapTiersReqJSON(t *testing.T) {
	body := `{"tiers":[{"max_amount":"3000","daily_cap":"600"},{"max_amount":"","daily_cap":"100000"}]}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rows, err := readDailyCapTiersReq(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].MaxAmount != "3000" || rows[0].DailyCap != "600" || rows[1].MaxAmount != "" || rows[1].DailyCap != "100000" {
		t.Fatalf("rows %+v", rows)
	}
}

func TestReadDailyCapTiersReqFormBrackets(t *testing.T) {
	form := url.Values{}
	form.Set("tiers[0][max_amount]", "3000")
	form.Set("tiers[0][daily_cap]", "600")
	form.Set("tiers[1][max_amount]", "")
	form.Set("tiers[1][daily_cap]", "100000")
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rows, err := readDailyCapTiersReq(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].MaxAmount != "3000" || rows[1].DailyCap != "100000" {
		t.Fatalf("rows %+v", rows)
	}
}

func TestReadDailyCapTiersReqFormJSON(t *testing.T) {
	form := url.Values{}
	form.Set("tiers", `[{"max_amount":"3000","daily_cap":"600"},{"max_amount":"","daily_cap":"100000"}]`)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rows, err := readDailyCapTiersReq(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].DailyCap != "600" || rows[1].MaxAmount != "" {
		t.Fatalf("rows %+v", rows)
	}
}

func TestReadDailyCapTiersReqInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`tiers[0][max_amount]=3000`))
	req.Header.Set("Content-Type", "application/json")
	if _, err := readDailyCapTiersReq(req); !errors.Is(err, biz.ErrConfigInvalid) {
		t.Fatalf("want invalid, got %v", err)
	}
}
