package ethtx

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/sha3"
)

const transferSelector = "a9059cbb"
const balanceOfSelector = "70a08231"

// ParsePrivateKey 解析 0x 可选的 32 字节私钥。
func ParsePrivateKey(hexKey string) (*secp256k1.PrivateKey, error) {
	s := strings.TrimSpace(hexKey)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 32 {
		return nil, fmt.Errorf("invalid private key")
	}
	if new(big.Int).SetBytes(b).Sign() == 0 {
		return nil, fmt.Errorf("invalid private key")
	}
	return secp256k1.PrivKeyFromBytes(b), nil
}

// AddressFromKey 由私钥得到小写 0x 地址。
func AddressFromKey(key *secp256k1.PrivateKey) string {
	if key == nil {
		return ""
	}
	uncompressed := key.PubKey().SerializeUncompressed()
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write(uncompressed[1:])
	sum := h.Sum(nil)
	return "0x" + hex.EncodeToString(sum[12:])
}

// TokenRaw 把业务金额（最多 8 位小数）换成代币最小单位。
func TokenRaw(amount decimal.Decimal, decimals int32) (*big.Int, error) {
	if decimals < 0 || decimals > 36 {
		return nil, fmt.Errorf("bad decimals")
	}
	if !amount.IsPositive() {
		return nil, fmt.Errorf("amount must be positive")
	}
	scale := decimal.New(1, decimals)
	raw := amount.Mul(scale).Truncate(0)
	if !raw.IsPositive() {
		return nil, fmt.Errorf("amount too small")
	}
	return raw.BigInt(), nil
}

// ERC20TransferData 构造 transfer(to,amount) calldata。
func ERC20TransferData(to string, amount *big.Int) ([]byte, error) {
	to = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(to)), "0x")
	if len(to) != 40 {
		return nil, fmt.Errorf("invalid to address")
	}
	if amount == nil || amount.Sign() <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}
	sel, _ := hex.DecodeString(transferSelector)
	addr, err := hex.DecodeString(to)
	if err != nil {
		return nil, err
	}
	amt := amount.Bytes()
	if len(amt) > 32 {
		return nil, fmt.Errorf("amount overflow")
	}
	out := make([]byte, 4+32+32)
	copy(out[0:4], sel)
	copy(out[4+12:4+32], addr)
	copy(out[4+32+32-len(amt):], amt)
	return out, nil
}

func padAddress(addr string) ([]byte, error) {
	addr = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(addr)), "0x")
	if len(addr) != 40 {
		return nil, fmt.Errorf("invalid address")
	}
	b, err := hex.DecodeString(addr)
	if err != nil || len(b) != 20 {
		return nil, fmt.Errorf("invalid address")
	}
	out := make([]byte, 32)
	copy(out[12:], b)
	return out, nil
}

// ERC20BalanceOfData 构造 balanceOf(holder) calldata。
func ERC20BalanceOfData(holder string) ([]byte, error) {
	sel, _ := hex.DecodeString(balanceOfSelector)
	padded, err := padAddress(holder)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 4+32)
	copy(out[0:4], sel)
	copy(out[4:], padded)
	return out, nil
}

// SignLegacyTx 签 EIP-155 legacy 交易，返回 raw tx（无 0x）。
func SignLegacyTx(key *secp256k1.PrivateKey, chainID, nonce, gasPrice, gasLimit *big.Int, to string, value *big.Int, data []byte) ([]byte, error) {
	if key == nil || chainID == nil || nonce == nil || gasPrice == nil || gasLimit == nil {
		return nil, fmt.Errorf("missing tx field")
	}
	to = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(to)), "0x")
	toBytes, err := hex.DecodeString(to)
	if err != nil || len(toBytes) != 20 {
		return nil, fmt.Errorf("invalid to")
	}
	if value == nil {
		value = big.NewInt(0)
	}
	if data == nil {
		data = []byte{}
	}
	hashRLP := rlpList(
		rlpUint(nonce),
		rlpUint(gasPrice),
		rlpUint(gasLimit),
		rlpBytes(toBytes),
		rlpUint(value),
		rlpBytes(data),
		rlpUint(chainID),
		rlpUint(big.NewInt(0)),
		rlpUint(big.NewInt(0)),
	)
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write(hashRLP)
	digest := h.Sum(nil)
	r, s, recID, err := signDigest(key, digest)
	if err != nil {
		return nil, err
	}
	v := new(big.Int).Mul(chainID, big.NewInt(2))
	v.Add(v, big.NewInt(int64(35+recID)))
	return rlpList(
		rlpUint(nonce),
		rlpUint(gasPrice),
		rlpUint(gasLimit),
		rlpBytes(toBytes),
		rlpUint(value),
		rlpBytes(data),
		rlpUint(v),
		rlpUint(r),
		rlpUint(s),
	), nil
}

func signDigest(key *secp256k1.PrivateKey, hash []byte) (r, s *big.Int, recID byte, err error) {
	compact := ecdsa.SignCompact(key, hash, false)
	if len(compact) != 65 {
		return nil, nil, 0, fmt.Errorf("sign failed")
	}
	header := compact[0]
	if header >= 27 {
		header -= 27
	}
	if header > 3 {
		return nil, nil, 0, fmt.Errorf("bad recovery id")
	}
	return new(big.Int).SetBytes(compact[1:33]), new(big.Int).SetBytes(compact[33:65]), header, nil
}

func rlpUint(n *big.Int) []byte {
	if n == nil || n.Sign() == 0 {
		return []byte{0x80}
	}
	return rlpBytes(n.Bytes())
}

func rlpBytes(b []byte) []byte {
	if len(b) == 1 && b[0] < 0x80 {
		return b
	}
	return append(rlpLen(0x80, len(b)), b...)
}

func rlpList(items ...[]byte) []byte {
	n := 0
	for _, it := range items {
		n += len(it)
	}
	return append(rlpLen(0xc0, n), join(items)...)
}

func rlpLen(offset byte, n int) []byte {
	if n <= 55 {
		return []byte{offset + byte(n)}
	}
	lenBytes := big.NewInt(int64(n)).Bytes()
	return append([]byte{offset + 55 + byte(len(lenBytes))}, lenBytes...)
}

func join(items [][]byte) []byte {
	n := 0
	for _, it := range items {
		n += len(it)
	}
	out := make([]byte, 0, n)
	for _, it := range items {
		out = append(out, it...)
	}
	return out
}
