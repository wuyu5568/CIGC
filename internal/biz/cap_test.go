package biz

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
)

func TestCapForAmount_DefaultTiers(t *testing.T) {
	ResetRuntimeCapTiers()
	cases := []struct {
		paid string
		want string
	}{
		{"0", "0"},
		{"1", "600"},
		{"1000", "600"},
		{"2999.99999999", "600"},
		{"3000", "1800"},
		{"6000", "4000"},
		{"12000", "8000"},
		{"24000", "16000"},
		{"36000", "24000"},
		{"50000", "30000"},
		{"70000", "42000"},
		{"100000", "60000"},
		{"160000", "100000"},
		{"200000", "100000"},
	}
	for _, tc := range cases {
		got := CapForAmount(decimal.RequireFromString(tc.paid))
		want := decimal.RequireFromString(tc.want)
		if !got.Equal(want) {
			t.Fatalf("paid=%s got=%s want=%s", tc.paid, got, want)
		}
	}
}

func TestCapForAmount_CustomTiers(t *testing.T) {
	t.Cleanup(ResetRuntimeCapTiers)
	tiers, err := NormalizeCapTiersJSON([]CapTierJSON{
		{MaxAmount: "1000", DailyCap: "100"},
		{MaxAmount: "", DailyCap: "900"},
	})
	if err != nil {
		t.Fatal(err)
	}
	SetRuntimeCapTiers(tiers)
	if got := CapForAmount(decimal.RequireFromString("999")); !got.Equal(decimal.RequireFromString("100")) {
		t.Fatalf("got %s", got)
	}
	if got := CapForAmount(decimal.RequireFromString("1000")); !got.Equal(decimal.RequireFromString("900")) {
		t.Fatalf("got %s", got)
	}
}

func TestNormalizeCapTiersJSON(t *testing.T) {
	if _, err := NormalizeCapTiersJSON(nil); err != ErrConfigInvalid {
		t.Fatalf("empty: %v", err)
	}
	if _, err := NormalizeCapTiersJSON([]CapTierJSON{{MaxAmount: "3000", DailyCap: "600"}}); err != ErrConfigInvalid {
		t.Fatalf("no unbounded: %v", err)
	}
	if _, err := NormalizeCapTiersJSON([]CapTierJSON{
		{MaxAmount: "6000", DailyCap: "1800"},
		{MaxAmount: "3000", DailyCap: "600"},
		{MaxAmount: "", DailyCap: "100000"},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := NormalizeCapTiersJSON([]CapTierJSON{
		{MaxAmount: "6000", DailyCap: "1800"},
		{MaxAmount: "3000", DailyCap: "600"},
		{MaxAmount: "", DailyCap: "100000"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || !got[0].Max.Equal(decimal.RequireFromString("3000")) || !got[2].Unbounded {
		t.Fatalf("%+v", got)
	}
}

func TestConfigUseCase_SaveDailyCapTiers(t *testing.T) {
	t.Cleanup(ResetRuntimeCapTiers)
	repo := &memConfigs{}
	uc := NewConfigUseCase(repo)
	rows, err := uc.SaveDailyCapTiers(context.Background(), []CapTierJSON{
		{MaxAmount: "2000", DailyCap: "50"},
		{MaxAmount: "", DailyCap: "80"},
	})
	if err != nil || len(rows) != 2 {
		t.Fatalf("%+v %v", rows, err)
	}
	if CapForAmount(decimal.RequireFromString("1999")).String() != "50" {
		t.Fatalf("runtime %s", CapForAmount(decimal.RequireFromString("1999")))
	}
	got := uc.DailyCapTiers(context.Background())
	if len(got) != 2 || got[0].MaxAmount != "2000" {
		t.Fatalf("%+v", got)
	}
}
