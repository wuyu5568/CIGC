package biz

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"

	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
)

const (
	ConfigDailyCapTiers     = "daily_cap_tiers"
	configDailyCapTiersName = "日封顶档位"
	maxCapTiers             = 20
)

// CapTier 一档日封顶：金额小于 Max 时套 Cap；Unbounded 表示以上全部。
type CapTier struct {
	Max       decimal.Decimal
	Unbounded bool
	Cap       decimal.Decimal
}

// CapTierJSON 管理端 / 前端档位行。
type CapTierJSON struct {
	MaxAmount string `json:"max_amount"`
	DailyCap  string `json:"daily_cap"`
}

var (
	capTiersMu sync.RWMutex
	capTiers   []CapTier
)

func defaultCapTiers() []CapTier {
	return []CapTier{
		{Max: decimal.RequireFromString("3000"), Cap: decimal.RequireFromString("600")},
		{Max: decimal.RequireFromString("6000"), Cap: decimal.RequireFromString("1800")},
		{Max: decimal.RequireFromString("12000"), Cap: decimal.RequireFromString("4000")},
		{Max: decimal.RequireFromString("24000"), Cap: decimal.RequireFromString("8000")},
		{Max: decimal.RequireFromString("36000"), Cap: decimal.RequireFromString("16000")},
		{Max: decimal.RequireFromString("50000"), Cap: decimal.RequireFromString("24000")},
		{Max: decimal.RequireFromString("70000"), Cap: decimal.RequireFromString("30000")},
		{Max: decimal.RequireFromString("100000"), Cap: decimal.RequireFromString("42000")},
		{Max: decimal.RequireFromString("160000"), Cap: decimal.RequireFromString("60000")},
		{Unbounded: true, Cap: decimal.RequireFromString("100000")},
	}
}

// SetRuntimeCapTiers 写入进程内档位；空切片回落默认。
func SetRuntimeCapTiers(tiers []CapTier) {
	capTiersMu.Lock()
	defer capTiersMu.Unlock()
	if len(tiers) == 0 {
		capTiers = nil
		return
	}
	capTiers = append([]CapTier(nil), tiers...)
}

// ResetRuntimeCapTiers 测试用：清缓存，回落默认档。
func ResetRuntimeCapTiers() {
	SetRuntimeCapTiers(nil)
}

// CapForAmount 按结算金额套日封顶。无金额为 0。
func CapForAmount(paid decimal.Decimal) decimal.Decimal {
	capTiersMu.RLock()
	tiers := capTiers
	capTiersMu.RUnlock()
	if len(tiers) == 0 {
		tiers = defaultCapTiers()
	}
	return matchCapTiers(paid, tiers)
}

func matchCapTiers(paid decimal.Decimal, tiers []CapTier) decimal.Decimal {
	paid = money.Round(paid)
	if !paid.IsPositive() {
		return decimal.Zero
	}
	var last decimal.Decimal
	for _, t := range tiers {
		last = t.Cap
		if t.Unbounded || paid.LessThan(t.Max) {
			return t.Cap
		}
	}
	return last
}

// CapTiersToJSON 输出给前端的档位表。
func CapTiersToJSON(tiers []CapTier) []CapTierJSON {
	if len(tiers) == 0 {
		tiers = defaultCapTiers()
	}
	out := make([]CapTierJSON, 0, len(tiers))
	for _, t := range tiers {
		row := CapTierJSON{DailyCap: money.Display(t.Cap)}
		if !t.Unbounded {
			row.MaxAmount = money.Display(t.Max)
		}
		out = append(out, row)
	}
	return out
}

// ParseCapTiersJSON 解析并校验档位表。
func ParseCapTiersJSON(raw string) ([]CapTier, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultCapTiers(), nil
	}
	var rows []CapTierJSON
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil, ErrConfigInvalid
	}
	return NormalizeCapTiersJSON(rows)
}

// NormalizeCapTiersJSON 校验、排序档位：有上限的按金额升序，最后一档上限必须留空。
func NormalizeCapTiersJSON(rows []CapTierJSON) ([]CapTier, error) {
	if len(rows) == 0 || len(rows) > maxCapTiers {
		return nil, ErrConfigInvalid
	}
	var bounded []CapTier
	var unbounded []CapTier
	for _, row := range rows {
		cap, err := money.Parse(row.DailyCap)
		if err != nil || cap.IsNegative() {
			return nil, ErrConfigInvalid
		}
		maxRaw := strings.TrimSpace(row.MaxAmount)
		if maxRaw == "" {
			unbounded = append(unbounded, CapTier{Unbounded: true, Cap: cap})
			continue
		}
		max, err := money.Parse(maxRaw)
		if err != nil || !max.IsPositive() {
			return nil, ErrConfigInvalid
		}
		bounded = append(bounded, CapTier{Max: max, Cap: cap})
	}
	if len(unbounded) != 1 {
		return nil, ErrConfigInvalid
	}
	sort.Slice(bounded, func(i, j int) bool {
		return bounded[i].Max.LessThan(bounded[j].Max)
	})
	for i := 1; i < len(bounded); i++ {
		if !bounded[i-1].Max.LessThan(bounded[i].Max) {
			return nil, ErrConfigInvalid
		}
	}
	return append(bounded, unbounded[0]), nil
}

// LoadCapTiersFromRepo 从 business_configs 刷新进程档位；失败则回落默认。
func LoadCapTiersFromRepo(ctx context.Context, configs ConfigRepo) {
	if configs == nil {
		SetRuntimeCapTiers(nil)
		return
	}
	raw, err := configs.GetValue(ctx, ConfigDailyCapTiers)
	if err != nil {
		SetRuntimeCapTiers(nil)
		return
	}
	tiers, err := ParseCapTiersJSON(raw)
	if err != nil {
		SetRuntimeCapTiers(nil)
		return
	}
	SetRuntimeCapTiers(tiers)
}
