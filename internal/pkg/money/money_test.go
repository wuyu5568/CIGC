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
