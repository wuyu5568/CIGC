package data

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/sha3"
)

func methodID(sig string) []byte {
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write([]byte(sig))
	return h.Sum(nil)[:4]
}

func padUint256(n uint64) []byte {
	b := make([]byte, 32)
	new(big.Int).SetUint64(n).FillBytes(b)
	return b
}

func encodeCall(sig string, args ...uint64) []byte {
	out := append([]byte{}, methodID(sig)...)
	for _, a := range args {
		out = append(out, padUint256(a)...)
	}
	return out
}

func parseHexBytes(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	if s == "" {
		return nil, fmt.Errorf("empty hex")
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("bad hex: %w", err)
	}
	return b, nil
}

func decodeABIUint256(data []byte) (*big.Int, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("uint256 too short")
	}
	return new(big.Int).SetBytes(data[:32]), nil
}

func decodeABIWordArray(data []byte) ([][]byte, error) {
	if len(data) < 64 {
		return nil, fmt.Errorf("dynamic array too short")
	}
	off := new(big.Int).SetBytes(data[:32])
	if !off.IsUint64() {
		return nil, fmt.Errorf("dynamic array bad offset")
	}
	offset := int(off.Uint64())
	if offset < 0 || offset+32 > len(data) {
		return nil, fmt.Errorf("dynamic array bad offset")
	}
	nBig := new(big.Int).SetBytes(data[offset : offset+32])
	if !nBig.IsUint64() {
		return nil, fmt.Errorf("dynamic array too large")
	}
	n := int(nBig.Uint64())
	start := offset + 32
	need := start + n*32
	if n < 0 || need > len(data) {
		return nil, fmt.Errorf("dynamic array truncated")
	}
	out := make([][]byte, n)
	for i := 0; i < n; i++ {
		word := make([]byte, 32)
		copy(word, data[start+i*32:start+i*32+32])
		out[i] = word
	}
	return out, nil
}

func decodeABIAddressArray(data []byte) ([]string, error) {
	words, err := decodeABIWordArray(data)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = "0x" + hex.EncodeToString(w[12:])
	}
	return out, nil
}

func decodeABIUint256Array(data []byte) ([]*big.Int, error) {
	words, err := decodeABIWordArray(data)
	if err != nil {
		return nil, err
	}
	out := make([]*big.Int, len(words))
	for i, w := range words {
		out[i] = new(big.Int).SetBytes(w)
	}
	return out, nil
}
