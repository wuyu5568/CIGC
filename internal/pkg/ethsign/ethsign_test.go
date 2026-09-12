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

func TestTextHashLen(t *testing.T) {
	h := TextHash([]byte("hi"))
	if len(h) != 32 {
		t.Fatalf("len=%d", len(h))
	}
}
