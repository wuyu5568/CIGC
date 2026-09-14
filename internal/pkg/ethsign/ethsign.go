package ethsign

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"golang.org/x/crypto/sha3"
)

// NormalizeAddress 将以太坊地址规范为小写 0x + 40 hex。
func NormalizeAddress(addr string) (string, bool) {
	addr = strings.TrimSpace(addr)
	if !strings.HasPrefix(addr, "0x") && !strings.HasPrefix(addr, "0X") {
		return "", false
	}
	body := addr[2:]
	if len(body) != 40 {
		return "", false
	}
	for _, c := range body {
		if !isHex(c) {
			return "", false
		}
	}
	return "0x" + strings.ToLower(body), true
}

// ChecksumAddress 返回 EIP-55 校验和地址；非法输入返回空串。
func ChecksumAddress(addr string) string {
	n, ok := NormalizeAddress(addr)
	if !ok {
		return ""
	}
	hexBody := n[2:]
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write([]byte(hexBody))
	sum := h.Sum(nil)
	out := []byte("0x")
	for i := 0; i < 40; i++ {
		c := hexBody[i]
		nibble := sum[i/2]
		if i%2 == 0 {
			nibble >>= 4
		} else {
			nibble &= 0x0f
		}
		if nibble >= 8 && c >= 'a' && c <= 'f' {
			c -= 'a' - 'A'
		}
		out = append(out, c)
	}
	return string(out)
}

func isHex(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// TextHash 计算 personal_sign 用的 Ethereum Signed Message 哈希。
func TextHash(message []byte) []byte {
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(message))
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write([]byte(prefix))
	_, _ = h.Write(message)
	return h.Sum(nil)
}

// RecoverAddress 从 personal_sign 签名恢复地址（小写 0x）。
func RecoverAddress(message []byte, sig []byte) (string, error) {
	if len(sig) != 65 {
		return "", fmt.Errorf("signature length")
	}
	hash := TextHash(message)
	header := sig[64]
	if header >= 27 {
		header -= 27
	}
	if header > 3 {
		return "", fmt.Errorf("invalid recovery id")
	}
	compact := make([]byte, 65)
	compact[0] = header + 27
	copy(compact[1:], sig[:64])
	pub, _, err := ecdsa.RecoverCompact(compact, hash)
	if err != nil {
		compact[0] = header + 31
		pub, _, err = ecdsa.RecoverCompact(compact, hash)
		if err != nil {
			return "", err
		}
	}
	return pubkeyToAddress(pub), nil
}

func pubkeyToAddress(pub *secp256k1.PublicKey) string {
	uncompressed := pub.SerializeUncompressed()
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write(uncompressed[1:])
	sum := h.Sum(nil)
	return "0x" + hex.EncodeToString(sum[12:])
}

// DecodeSignature 解析 0x 开头的 65 字节签名。
func DecodeSignature(signature string) ([]byte, error) {
	signature = strings.TrimSpace(signature)
	signature = strings.TrimPrefix(signature, "0x")
	signature = strings.TrimPrefix(signature, "0X")
	b, err := hex.DecodeString(signature)
	if err != nil {
		return nil, err
	}
	if len(b) != 65 {
		return nil, fmt.Errorf("signature length")
	}
	return b, nil
}
