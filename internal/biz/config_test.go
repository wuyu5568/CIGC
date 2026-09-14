package biz

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
)

func TestNormalizeConfigValue(t *testing.T) {
	v, err := NormalizeConfigValue(ConfigDirectRate, "0.15")
	if err != nil || v != "0.15" {
		t.Fatalf("%s %v", v, err)
	}
	if _, err := NormalizeConfigValue(ConfigDirectRate, "1.2"); err != ErrConfigInvalid {
		t.Fatalf("rate>1: %v", err)
	}
	if _, err := NormalizeConfigValue(ConfigMinWithdraw, "0"); err != ErrConfigInvalid {
		t.Fatalf("min=0: %v", err)
	}
	v, err = NormalizeConfigValue(ConfigWithdrawFeeRate, "0")
	if err != nil || v != "0" {
		t.Fatalf("fee 0: %s %v", v, err)
	}
	if _, err := NormalizeConfigValue(ConfigWithdrawFeeRate, "1.2"); err != ErrConfigInvalid {
		t.Fatalf("fee>1: %v", err)
	}
	v, err = NormalizeConfigValue(ConfigWithdrawDaily, "0")
	if err != nil || v != "0" {
		t.Fatalf("daily 0: %s %v", v, err)
	}
	if _, err := NormalizeConfigValue(ConfigWithdrawDaily, "-1"); err != ErrConfigInvalid {
		t.Fatalf("daily<0: %v", err)
	}
	if _, err := NormalizeConfigValue(ConfigWithdrawDailyIspay, "0"); err != nil {
		t.Fatal(err)
	}
	v, err = NormalizeConfigValue(ConfigIspayPrice, "1800")
	if err != nil || v != "1800" {
		t.Fatalf("price %s %v", v, err)
	}
	if _, err := NormalizeConfigValue("unknown", "1"); err != ErrConfigForbidden {
		t.Fatalf("unknown: %v", err)
	}
}

func TestConfigUseCase_UpdateAndSpot(t *testing.T) {
	repo := &memConfigs{
		rows: []*BusinessConfig{
			{ID: 1, Key: ConfigDirectRate, Name: "直推", Value: "0.10", SortOrder: 10},
			{ID: 2, Key: ConfigIspayPrice, Name: "现价", Value: "2000", SortOrder: 50},
		},
		ispayPrice: "2000",
	}
	uc := NewConfigUseCase(repo)
	list, err := uc.List(context.Background())
	if err != nil || len(list) != 2 {
		t.Fatalf("%+v %v", list, err)
	}
	got, err := uc.Update(context.Background(), 1, "0.12")
	if err != nil || got.Value != "0.12" {
		t.Fatalf("%+v %v", got, err)
	}
	if v, _ := repo.GetValue(context.Background(), ConfigDirectRate); v != "0.12" {
		t.Fatalf("stored %s", v)
	}
	if _, err := uc.Update(context.Background(), 99, "1"); err != ErrConfigNotFound {
		t.Fatalf("missing: %v", err)
	}
	if !uc.Spot(context.Background()).Equal(decimal.RequireFromString("2000")) {
		t.Fatalf("spot default %s", uc.Spot(context.Background()))
	}
	if _, err := uc.Update(context.Background(), 2, "1800"); err != nil {
		t.Fatal(err)
	}
	if !uc.Spot(context.Background()).Equal(decimal.RequireFromString("1800")) {
		t.Fatalf("spot %s", uc.Spot(context.Background()))
	}
}
