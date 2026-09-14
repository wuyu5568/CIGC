package ethtx

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/sha3"
)

func TestTokenRawAndCalldata(t *testing.T) {
	raw, err := TokenRaw(decimal.RequireFromString("1.5"), 18)
	if err != nil {
		t.Fatal(err)
	}
	want := new(big.Int)
	want.SetString("1500000000000000000", 10)
	if raw.Cmp(want) != 0 {
		t.Fatalf("raw=%s", raw)
	}
	data, err := ERC20TransferData("0x1111111111111111111111111111111111111111", raw)
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(data[:4]) != transferSelector {
		t.Fatalf("sel=%x", data[:4])
	}
	if hex.EncodeToString(data[16:36]) != "1111111111111111111111111111111111111111" {
		t.Fatalf("to=%x", data[4:36])
	}
	balData, err := ERC20BalanceOfData("0x1111111111111111111111111111111111111111")
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(balData[:4]) != balanceOfSelector {
		t.Fatalf("balanceOf sel=%x", balData[:4])
	}
	if hex.EncodeToString(balData[16:36]) != "1111111111111111111111111111111111111111" {
		t.Fatalf("holder=%x", balData[4:36])
	}
}

func TestSignLegacyTxRecovers(t *testing.T) {
	key, err := ParsePrivateKey("0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318")
	if err != nil {
		t.Fatal(err)
	}
	from := AddressFromKey(key)
	if from != "0x2c7536e3605d9c16a7a3d7b1898e529396a65c23" {
		t.Fatalf("from=%s", from)
	}
	data, err := ERC20TransferData("0x2222222222222222222222222222222222222222", big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := SignLegacyTx(key, big.NewInt(56), big.NewInt(1), big.NewInt(5e9), big.NewInt(80000),
		"0x55d398326f99059ff775485246999027b3197955", big.NewInt(0), data)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 100 {
		t.Fatalf("raw too short %d", len(raw))
	}
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write(rlpList(
		rlpUint(big.NewInt(1)),
		rlpUint(big.NewInt(5e9)),
		rlpUint(big.NewInt(80000)),
		mustHex("55d398326f99059ff775485246999027b3197955"),
		rlpUint(big.NewInt(0)),
		rlpBytes(data),
		rlpUint(big.NewInt(56)),
		rlpUint(big.NewInt(0)),
		rlpUint(big.NewInt(0)),
	))
	digest := h.Sum(nil)
	r, s, recID, err := signDigest(key, digest)
	if err != nil {
		t.Fatal(err)
	}
	compact := make([]byte, 65)
	compact[0] = recID + 27
	copy(compact[1:33], pad32(r.Bytes()))
	copy(compact[33:65], pad32(s.Bytes()))
	pub, _, err := ecdsa.RecoverCompact(compact, digest)
	if err != nil {
		t.Fatal(err)
	}
	uncompressed := pub.SerializeUncompressed()
	kh := sha3.NewLegacyKeccak256()
	_, _ = kh.Write(uncompressed[1:])
	sum := kh.Sum(nil)
	got := "0x" + hex.EncodeToString(sum[12:])
	if got != from {
		t.Fatalf("recover %s want %s", got, from)
	}
}

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return rlpBytes(b)
}

func pad32(b []byte) []byte {
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}
