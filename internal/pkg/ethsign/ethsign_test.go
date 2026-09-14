package ethsign

import "testing"

func TestNormalizeAddress(t *testing.T) {
	got, ok := NormalizeAddress("0xAAAAAAAAAAaaaaaaaaAAAAAAAAAAaaaaaaaaAAAA")
	if !ok || got != "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	if _, ok := NormalizeAddress("0xabc"); ok {
		t.Fatal("short address should fail")
	}
}

func TestChecksumAddress(t *testing.T) {
	got := ChecksumAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
	if got != "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed" {
		t.Fatalf("got %q", got)
	}
	genesis := ChecksumAddress("0x8bd86ad98d9fa366e52cfb5c08a3e33f5f412fb1")
	if genesis != "0x8Bd86ad98D9fA366E52CFB5c08a3e33f5f412Fb1" {
		t.Fatalf("genesis checksum %q", genesis)
	}
}

func TestTextHashLen(t *testing.T) {
	h := TextHash([]byte("hi"))
	if len(h) != 32 {
		t.Fatalf("len=%d", len(h))
	}
}
