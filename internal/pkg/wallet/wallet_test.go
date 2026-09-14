package wallet

import "testing"

func TestNormalizeReceiveAddress_Keeps39Hex(t *testing.T) {
	got := NormalizeReceiveAddress("0x573ffd2739a3be7054ea6c3bb3581349f0745e8")
	if got != "0x573ffd2739a3be7054ea6c3bb3581349f0745e8" {
		t.Fatalf("got %q", got)
	}
	if ReceiveKey(got) != "0x0573ffd2739a3be7054ea6c3bb3581349f0745e8" {
		t.Fatalf("key %q", ReceiveKey(got))
	}
	if NormalizeOrEmpty("0x573ffd2739a3be7054ea6c3bb3581349f0745e8") != "" {
		t.Fatal("user wallets must stay 40 hex")
	}
}

func TestNormalizeReceiveAddress_Standard(t *testing.T) {
	got := NormalizeReceiveAddress("0xA1e54373034aae3c00df1b9b89b20d2df55e2cad")
	if got != "0xa1e54373034aae3c00df1b9b89b20d2df55e2cad" {
		t.Fatalf("got %q", got)
	}
}
