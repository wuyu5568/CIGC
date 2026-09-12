package conf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyEnvOverrides_Payout(t *testing.T) {
	t.Setenv(EnvPayoutEnabled, "1")
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
	if err := ValidateAppSafety(&App{PayoutEnabled: true, PayoutMaxUSDT: 1}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAppSafety(&App{PayoutEnabled: false, PayoutMaxUSDT: 0}); err != nil {
		t.Fatal(err)
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
