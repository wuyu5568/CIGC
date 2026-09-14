package biz

import (
	"fmt"
	"strings"

	"github.com/cigc/app/internal/conf"
	"github.com/cigc/app/internal/pkg/money"
	"github.com/cigc/app/internal/pkg/wallet"
	"github.com/shopspring/decimal"
)

// ReceiveShare 是一条收款地址与分配百分比（75 表示 75%）。
type ReceiveShare struct {
	Address string
	Percent decimal.Decimal
}

// ReceivePayItem 是某笔订单金额按比例拆出的应付份额。
type ReceivePayItem struct {
	Address string
	Percent decimal.Decimal
	Amount  decimal.Decimal
}

// ParseReceiveShares 读配置：多地址列表优先，否则单地址视为 100%。
func ParseReceiveShares(app *conf.App) ([]ReceiveShare, error) {
	if app == nil {
		return nil, nil
	}
	raw := app.ReceiveAddresses
	if len(raw) == 0 && strings.TrimSpace(app.ReceiveAddress) != "" {
		raw = []conf.ReceiveShare{{Address: app.ReceiveAddress, Percent: "100"}}
	}
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]ReceiveShare, 0, len(raw))
	sum := decimal.Zero
	seen := map[string]struct{}{}
	for i, it := range raw {
		addr := wallet.NormalizeReceiveAddress(it.Address)
		if addr == "" {
			return nil, fmt.Errorf("receive_addresses[%d]: invalid address", i)
		}
		if _, ok := seen[addr]; ok {
			return nil, fmt.Errorf("receive_addresses[%d]: duplicate address", i)
		}
		seen[addr] = struct{}{}
		pct, err := parsePercent(it.Percent)
		if err != nil {
			return nil, fmt.Errorf("receive_addresses[%d]: %w", i, err)
		}
		out = append(out, ReceiveShare{Address: addr, Percent: pct})
		sum = sum.Add(pct)
	}
	if !money.Round(sum).Equal(decimal.NewFromInt(100)) {
		return nil, fmt.Errorf("receive_addresses percents must sum to 100, got %s", money.Round(sum))
	}
	return out, nil
}

func parsePercent(s string) (decimal.Decimal, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	s = strings.TrimSpace(s)
	if s == "" {
		return decimal.Zero, fmt.Errorf("empty percent")
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, fmt.Errorf("bad percent %q", s)
	}
	d = money.Round(d)
	if !d.IsPositive() {
		return decimal.Zero, fmt.Errorf("percent must be positive")
	}
	return d, nil
}

// AllocateAmounts 按百分比拆金额；最后一笔吃余数，保证加总等于 total。
func AllocateAmounts(total decimal.Decimal, shares []ReceiveShare) []ReceivePayItem {
	total = money.Round(total)
	out := make([]ReceivePayItem, len(shares))
	if len(shares) == 0 {
		return out
	}
	hundred := decimal.NewFromInt(100)
	used := decimal.Zero
	for i, s := range shares {
		var amt decimal.Decimal
		if i == len(shares)-1 {
			amt = money.Round(total.Sub(used))
		} else {
			amt = money.Round(total.Mul(s.Percent).Div(hundred))
			used = used.Add(amt)
		}
		if amt.IsNegative() {
			amt = decimal.Zero
		}
		out[i] = ReceivePayItem{Address: s.Address, Percent: s.Percent, Amount: amt}
	}
	return out
}

func receiveSet(shares []ReceiveShare) map[string]ReceiveShare {
	m := make(map[string]ReceiveShare, len(shares))
	for _, s := range shares {
		m[wallet.ReceiveKey(s.Address)] = s
	}
	return m
}

func shareOf(byAddr map[string]ReceiveShare, addr string) (ReceiveShare, bool) {
	s, ok := byAddr[wallet.ReceiveKey(addr)]
	return s, ok
}

func firstReceive(shares []ReceiveShare) string {
	if len(shares) == 0 {
		return ""
	}
	return shares[0].Address
}
