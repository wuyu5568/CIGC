package conf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStripCronExpr_QuotedDockerDefault(t *testing.T) {
	if got := stripCronExpr(`"* * * * *"`); got != "* * * * *" {
		t.Fatalf("got %q", got)
	}
	if got := stripCronExpr(`'* * * * *'`); got != "* * * * *" {
		t.Fatalf("got %q", got)
	}
	if got := stripCronExpr("0 0 * * *"); got != "0 0 * * *" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyEnvOverrides_QuotedDepositCron(t *testing.T) {
	t.Setenv(EnvDepositCron, `"* * * * *"`)
	t.Setenv(EnvPayoutCron, `'* * * * *'`)
	bc := &Bootstrap{}
	applyEnvOverrides(bc)
	if bc.App.DepositCron != "* * * * *" {
		t.Fatalf("deposit cron: got %q", bc.App.DepositCron)
	}
	if bc.App.PayoutCron != "* * * * *" {
		t.Fatalf("payout cron: got %q", bc.App.PayoutCron)
	}
}

func TestApplyEnvOverrides_Payout(t *testing.T) {
	t.Setenv(EnvPayoutEnabled, "1")
	t.Setenv(EnvPayoutCron, "*/5 * * * *")
	t.Setenv(EnvHotWalletKey, "aabb")
	t.Setenv(EnvBscRPC, "https://example.invalid/")

	bc := &Bootstrap{}
	applyEnvOverrides(bc)

	if !bc.App.PayoutEnabled {
		t.Fatal("expected payout enabled from env")
	}
	if bc.App.HotWalletKey != "aabb" {
		t.Fatalf("hot key: got %q", bc.App.HotWalletKey)
	}
	if bc.App.BscRPC != "https://example.invalid/" {
		t.Fatalf("rpc: got %q", bc.App.BscRPC)
	}
	if bc.App.PayoutCron != "*/5 * * * *" {
		t.Fatalf("cron: got %q", bc.App.PayoutCron)
	}
}

func TestApplyEnvOverrides_SecretsAndDSN(t *testing.T) {
	t.Setenv(EnvHTTPAddr, "0.0.0.0:9000")
	t.Setenv(EnvDatabaseDSN, "user:pass@tcp(mysql:3306)/cigc")
	t.Setenv(EnvJWTKey, "jwt-from-env")
	t.Setenv(EnvAdminUsername, "root-admin")
	t.Setenv(EnvAdminPassword, "secret")
	t.Setenv(EnvGenesisAddress, "0xabc")

	bc := &Bootstrap{}
	applyEnvOverrides(bc)

	if bc.Server.HTTP.Addr != "0.0.0.0:9000" {
		t.Fatalf("addr: got %q", bc.Server.HTTP.Addr)
	}
	if bc.Data.Database.Source != "user:pass@tcp(mysql:3306)/cigc" {
		t.Fatalf("dsn: got %q", bc.Data.Database.Source)
	}
	if bc.Auth.JWTKey != "jwt-from-env" {
		t.Fatalf("jwt: got %q", bc.Auth.JWTKey)
	}
	if bc.Auth.AdminUsername != "root-admin" || bc.Auth.AdminPassword != "secret" {
		t.Fatalf("admin: %+v", bc.Auth)
	}
	if bc.App.GenesisAddress != "0xabc" {
		t.Fatalf("genesis: got %q", bc.App.GenesisAddress)
	}
}

func TestValidate_RequiresDSNAndJWT(t *testing.T) {
	if err := Validate(&Bootstrap{}); err == nil {
		t.Fatal("expected missing dsn error")
	}
	bc := &Bootstrap{}
	bc.Data.Database.Source = "dsn"
	if err := Validate(bc); err == nil {
		t.Fatal("expected missing jwt error")
	}
	bc.Auth.JWTKey = "k"
	if err := Validate(bc); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAppSafety_PayoutNeedsMax(t *testing.T) {
	if err := ValidateAppSafety(&App{PayoutEnabled: true, PayoutMaxUSDT: 0}); err == nil {
		t.Fatal("expected error when payout on without max")
	}
	full := &App{
		PayoutEnabled: true,
		PayoutMaxUSDT: 1,
		HotWalletKey:  "aa",
		BscRPC:        "https://example.invalid/",
		UsdtAddress:   "0x55d398326f99059fF775485246999027B3197955",
	}
	if err := ValidateAppSafety(full); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAppSafety(&App{PayoutEnabled: false, PayoutMaxUSDT: 0}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReceive_DefaultPercents(t *testing.T) {
	err := ValidateReceive(&App{ReceiveAddresses: []ReceiveShare{
		{Address: "0xa1e54373034aae3c00df1b9b89b20d2df55e2cad", Percent: "80"},
		{Address: "0xE7Da6c5D90f6a88fEEa228d5C9a0611c61F7500D", Percent: "10"},
		{Address: "0x623ecc54647605c220199f4d273cf9f43fddd5c1", Percent: "5"},
		{Address: "0x907D9173ab226C698C178981c4D135f8168dD6eb", Percent: "3"},
		{Address: "0x279F2B0B788b50c90ceCd74C9134D152083A87B7", Percent: "1.5"},
		{Address: "0xd3E7fE539c291010B8948Fe19372ac9223288109", Percent: "0.5"},
	}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestApplyEnvOverrides_IspayPayout(t *testing.T) {
	t.Setenv(EnvIspayAddress, "0xBF9b0594E110C381F2606961C78641a194999999")
	t.Setenv(EnvPayoutMaxIspay, "1000")
	bc := &Bootstrap{}
	applyEnvOverrides(bc)
	if bc.App.IspayAddress != "0xBF9b0594E110C381F2606961C78641a194999999" {
		t.Fatalf("ispay: got %q", bc.App.IspayAddress)
	}
	if bc.App.PayoutMaxIspay != 1000 {
		t.Fatalf("max ispay: got %v", bc.App.PayoutMaxIspay)
	}
}

func TestApplyEnvOverrides_BuyContract(t *testing.T) {
	t.Setenv(EnvBuyContract, "0xCb63733FB936c7B3f147C757D383645e55769bF3")
	bc := &Bootstrap{}
	applyEnvOverrides(bc)
	if bc.App.BuyContract != "0xCb63733FB936c7B3f147C757D383645e55769bF3" {
		t.Fatalf("buy: got %q", bc.App.BuyContract)
	}
}

func TestApplyEnvOverrides_ReceiveAddressesJSON(t *testing.T) {
	t.Setenv(EnvReceiveAddresses, `[{"address":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","percent":"60"},{"address":"0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","percent":"40"}]`)
	bc := &Bootstrap{}
	applyEnvOverrides(bc)
	if len(bc.App.ReceiveAddresses) != 2 || bc.App.ReceiveAddresses[0].Percent != "60" {
		t.Fatalf("%+v", bc.App.ReceiveAddresses)
	}
}

func TestEnvTruthy(t *testing.T) {
	for _, v := range []string{"1", "TRUE", "yes", "on"} {
		if !envTruthy(v) {
			t.Fatalf("%q should be truthy", v)
		}
	}
	for _, v := range []string{"0", "false", "", "no"} {
		if envTruthy(v) {
			t.Fatalf("%q should be falsy", v)
		}
	}
}

func TestLoad_EnvOverridesYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	body := []byte("server:\n  http:\n    addr: 127.0.0.1:8000\n    timeout: 10s\n" +
		"data:\n  database:\n    driver: mysql\n    source: yaml-dsn\n" +
		"auth:\n  jwt_key: yaml-jwt\n")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvDatabaseDSN, "env-dsn")
	t.Setenv(EnvJWTKey, "env-jwt")
	bc, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if bc.Data.Database.Source != "env-dsn" {
		t.Fatalf("dsn: got %q", bc.Data.Database.Source)
	}
	if bc.Auth.JWTKey != "env-jwt" {
		t.Fatalf("jwt: got %q", bc.Auth.JWTKey)
	}
}
