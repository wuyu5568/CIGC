package money

import "testing"

func TestRound_EightPlaces(t *testing.T) {
	got := MustParse("1.234567891").StringFixed(8)
	if got != "1.23456789" {
		t.Fatalf("got %s", got)
	}
}

func TestRound_Zero(t *testing.T) {
	if !Round(MustParse("0")).IsZero() {
		t.Fatal("expected zero")
	}
}

func TestDisplay_TrimZeros(t *testing.T) {
	if got := Display(MustParse("1000")); got != "1000" {
		t.Fatalf("int=%s", got)
	}
	if got := Display(MustParse("1000.50000000")); got != "1000.5" {
		t.Fatalf("frac=%s", got)
	}
	if got := Display(MustParse("33.33333000")); got != "33.33333" {
		t.Fatalf("keep=%s", got)
	}
}
