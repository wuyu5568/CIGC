package biz

import (
	"testing"

	"github.com/cigc/app/internal/conf"
	"github.com/shopspring/decimal"
)

func TestParseReceiveShares_DefaultListAsWritten(t *testing.T) {
	shares, err := ParseReceiveShares(&conf.App{
		ReceiveAddresses: []conf.ReceiveShare{
			{Address: "0xdf4cbc6c9c4f084b6e75177852c6a29f206b8fe1", Percent: "89"},
			{Address: "0x1c630cC605C96B1Bbcc1aCE704a7Ba0f259E6098", Percent: "5"},
			{Address: "0xa1e54373034aae3c00df1b9b89b20d2df55e2cad", Percent: "5"},
			{Address: "0xE7Da6c5D90f6a88fEEa228d5C9a0611c61F7500D", Percent: "1"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(shares) != 4 {
		t.Fatalf("len=%d", len(shares))
	}
	if shares[0].Address != "0xdf4cbc6c9c4f084b6e75177852c6a29f206b8fe1" {
		t.Fatalf("as-written: %s", shares[0].Address)
	}
	plan := AllocateAmounts(decimal.RequireFromString("1000"), shares)
	if len(plan) != 4 || !plan[0].Amount.Equal(decimal.RequireFromString("890")) {
		t.Fatalf("890: %+v", plan)
	}
	if !plan[1].Amount.Equal(decimal.RequireFromString("50")) || !plan[2].Amount.Equal(decimal.RequireFromString("50")) {
		t.Fatalf("mid: %+v", plan)
	}
	if !plan[3].Amount.Equal(decimal.RequireFromString("10")) {
		t.Fatalf("last: %s", plan[3].Amount)
	}
	sum := decimal.Zero
	for _, it := range plan {
		sum = sum.Add(it.Amount)
	}
	if !sum.Equal(decimal.RequireFromString("1000")) {
		t.Fatalf("sum=%s", sum)
	}
}

func TestParseReceiveShares_SingleFallback(t *testing.T) {
	shares, err := ParseReceiveShares(&conf.App{
		ReceiveAddress: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if err != nil || len(shares) != 1 || !shares[0].Percent.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("%+v %v", shares, err)
	}
}

func TestParseReceiveShares_SumMustBe100(t *testing.T) {
	_, err := ParseReceiveShares(&conf.App{
		ReceiveAddresses: []conf.ReceiveShare{
			{Address: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Percent: "60"},
			{Address: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Percent: "10"},
		},
	})
	if err == nil {
		t.Fatal("expected sum error")
	}
}
