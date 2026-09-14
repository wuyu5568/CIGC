package data

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"
)

func TestMethodID_BuySomething(t *testing.T) {
	a := methodID("getUserLength()")
	b := methodID("getUsersByIndex(uint256,uint256)")
	c := methodID("getUsersAmountByIndex(uint256,uint256)")
	if len(a) != 4 || len(b) != 4 || len(c) != 4 {
		t.Fatal("selector length")
	}
	if hex.EncodeToString(a) == hex.EncodeToString(b) || hex.EncodeToString(a) == hex.EncodeToString(c) {
		t.Fatal("selectors collided")
	}
	data := encodeCall("getUserLength()")
	if hex.EncodeToString(data) != hex.EncodeToString(a) {
		t.Fatalf("encodeCall prefix")
	}
}

func TestEncodeCall_UintArgs(t *testing.T) {
	data := encodeCall("getUsersByIndex(uint256,uint256)", 1, 3)
	if len(data) != 4+64 {
		t.Fatalf("len %d", len(data))
	}
	start := new(big.Int).SetBytes(data[4:36])
	end := new(big.Int).SetBytes(data[36:68])
	if start.Uint64() != 1 || end.Uint64() != 3 {
		t.Fatalf("args %s %s", start, end)
	}
}

func TestDecodeABIUint256AndArrays(t *testing.T) {
	raw, err := parseHexBytes("0x" + strings.Repeat("0", 62) + "2a")
	if err != nil {
		t.Fatal(err)
	}
	n, err := decodeABIUint256(raw)
	if err != nil || n.Uint64() != 42 {
		t.Fatalf("%v %v", n, err)
	}

	addr := strings.Repeat("b", 40)
	encoded := "0x" +
		strings.Repeat("0", 62) + "20" +
		strings.Repeat("0", 62) + "01" +
		strings.Repeat("0", 24) + addr
	addrs, err := decodeABIAddressArray(mustHex(t, encoded))
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 1 || addrs[0] != "0x"+addr {
		t.Fatalf("%v", addrs)
	}

	amt := "0x" +
		strings.Repeat("0", 62) + "20" +
		strings.Repeat("0", 62) + "01" +
		strings.Repeat("0", 63) + "5"
	amts, err := decodeABIUint256Array(mustHex(t, amt))
	if err != nil || len(amts) != 1 || amts[0].Uint64() != 5 {
		t.Fatalf("%v %v", amts, err)
	}
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := parseHexBytes(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
