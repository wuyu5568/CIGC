package money

import (
	"strings"

	"github.com/shopspring/decimal"
)

// Scale is the jinniu-compatible business amount scale (DECIMAL(36,8)).
const Scale int32 = 8

// Round applies bankers-style shopspring Round to 8 decimal places.
func Round(d decimal.Decimal) decimal.Decimal {
	return d.Round(Scale)
}

// Parse parses a decimal string and rounds to Scale.
func Parse(s string) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(strings.TrimSpace(s))
	if err != nil {
		return decimal.Zero, err
	}
	return Round(d), nil
}

// MustParse parses or panics (tests / static seeds only).
func MustParse(s string) decimal.Decimal {
	d, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}
