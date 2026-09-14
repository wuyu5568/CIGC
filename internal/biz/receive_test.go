package biz

import (
	"testing"

	"github.com/cigc/app/internal/conf"
	"github.com/shopspring/decimal"
)

func TestParseReceiveShares_DefaultListAsWritten(t *testing.T) {
	shares, err := ParseReceiveShares(&conf.App{
		ReceiveAddresses: []conf.ReceiveShare{
			{Address: "0xa1e54373034aae3c00df1b9b89b20d2df55e2cad", Percent: "80"},
			{Address: "0xE7Da6c5D90f6a88fEEa228d5C9a0611c61F7500D", Percent: "10"},
			{Address: "0x623ecc54647605c220199f4d273cf9f43fddd5c1", Percent: "5"},
			{Address: "0x907D9173ab226C698C178981c4D135f8168dD6eb", Percent: "3"},
			{Address: "0x279F2B0B788b50c90ceCd74C9134D152083A87B7", Percent: "1.5"},
			{Address: "0xd3E7fE539c291010B8948Fe19372ac9223288109", Percent: "0.5"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(shares) != 6 {
		t.Fatalf("len=%d", len(shares))
	}
	if shares[0].Address != "0xa1e54373034aae3c00df1b9b89b20d2df55e2cad" {
		t.Fatalf("as-written: %s", shares[0].Address)
	}
	plan := AllocateAmounts(decimal.RequireFromString("1000"), shares)
	want := []string{"800", "100", "50", "30", "15", "5"}
	if len(plan) != 6 {
		t.Fatalf("plan=%+v", plan)
	}
	for i, w := range want {
		if !plan[i].Amount.Equal(decimal.RequireFromString(w)) {
			t.Fatalf("share %d: got %s want %s", i, plan[i].Amount, w)
		}
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
