package wallet

import (
	"strings"

	"github.com/cigc/app/internal/pkg/ethsign"
)

// NormalizeAddress 将钱包地址规范为小写 0x hex。
func NormalizeAddress(addr string) (string, bool) {
	return ethsign.NormalizeAddress(addr)
}

// NormalizeOrEmpty 非法地址返回空串。
func NormalizeOrEmpty(addr string) string {
	a, ok := NormalizeAddress(addr)
	if !ok {
		return ""
	}
	return a
}

// NormalizeReceiveAddress 收款地址按原文保留：0x + hex 小写，不补位。
func NormalizeReceiveAddress(addr string) string {
	if a := NormalizeOrEmpty(addr); a != "" {
		return a
	}
	addr = strings.TrimSpace(addr)
	if len(addr) < 3 || !strings.EqualFold(addr[:2], "0x") {
		return ""
	}
	body := addr[2:]
	if len(body) == 0 || len(body) > 40 {
		return ""
	}
	for _, c := range body {
		if !isHex(c) {
			return ""
		}
	}
	return "0x" + strings.ToLower(body)
}

// ReceiveKey 扫链比对用的 40 hex 键；配置原文不变。
func ReceiveKey(addr string) string {
	a := NormalizeReceiveAddress(addr)
	if a == "" {
		return ""
	}
	body := a[2:]
	if len(body) >= 40 {
		return "0x" + body
	}
	return "0x" + strings.Repeat("0", 40-len(body)) + body
}

func isHex(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
