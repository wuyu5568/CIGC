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
	v, err = NormalizeConfigValue(ConfigMinWithdrawIspay, "0")
	if err != nil || v != "0" {
		t.Fatalf("ispay min 0: %s %v", v, err)
	}
	v, err = NormalizeConfigValue(ConfigWithdrawFeeIspay, "0.05")
	if err != nil || v != "0.05" {
		t.Fatalf("ispay fee: %s %v", v, err)
	}
	v, err = NormalizeConfigValue(ConfigManageGens, "4")
	if err != nil || v != "4" {
		t.Fatalf("gens: %s %v", v, err)
	}
	if _, err := NormalizeConfigValue(ConfigManageGens, "11"); err != ErrConfigInvalid {
		t.Fatalf("gens 11: %v", err)
	}
	if _, err := NormalizeConfigValue(ConfigManageGens, "3.5"); err != ErrConfigInvalid {
		t.Fatalf("gens 3.5: %v", err)
	}
	v, err = NormalizeConfigValue(ConfigOverflowHours, "72")
	if err != nil || v != "72" {
		t.Fatalf("overflow hours: %s %v", v, err)
	}
	if _, err := NormalizeConfigValue(ConfigOverflowHours, "0"); err != ErrConfigInvalid {
		t.Fatalf("overflow 0: %v", err)
	}
	if _, err := NormalizeConfigValue(ConfigOverflowHours, "721"); err != ErrConfigInvalid {
		t.Fatalf("overflow 721: %v", err)
	}
}

func TestConfigMeta_Groups(t *testing.T) {
	g, _, _ := ConfigMeta(ConfigDirectRate)
	if g != "奖励" {
		t.Fatalf("direct group %s", g)
	}
	g, _, _ = ConfigMeta(ConfigMinWithdrawIspay)
	if g != "提现" {
		t.Fatalf("ispay min group %s", g)
	}
	g, _, _ = ConfigMeta(ConfigOverflowHours)
	if g != "冻结" {
		t.Fatalf("overflow group %s", g)
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
