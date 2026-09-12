package wallet

import "github.com/cigc/app/internal/pkg/ethsign"

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
