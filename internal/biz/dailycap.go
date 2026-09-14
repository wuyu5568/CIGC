package biz

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
)

const OverflowClearHours = 72 // 次日 0:00 封账后再过 N 小时归零（默认 72h）

const (
	LedgerCapOverflowBurn     = "cap_overflow_burn"
	LedgerCapOverflowRelease  = "cap_overflow_release"
	overflowSourceInactive    = "inactive"
	overflowSourceAdmin       = "admin"
	remarkOverflowOrderUnlock = "overflow unlock by order cap"
	remarkOverflowClear72h    = "overflow clear 72h"
	remarkInactiveWrap        = "inactive freeze wrap"
	remarkAdminLockHold       = "admin adjust lock"
)

// CapOverflowHold 是一笔超过当日动态封顶的冻结；到期清除，不转可提现。
type CapOverflowHold struct {
	ID         uint64
	UserID     uint64
	Value      decimal.Decimal
	USDT       decimal.Decimal
	Ispay      decimal.Decimal
	SourceType string
	OrderID    *uint64
	SettleDate time.Time
	Remark     string
	CreatedAt  time.Time
	ExpiresAt  *time.Time
	ReleasedAt *time.Time
	BurnedAt   *time.Time
}

// DailyCapRepo 当日动态已用额度与超额冻结。
type DailyCapRepo interface {
	TakeUnder(ctx context.Context, userID uint64, settleDate time.Time, cap, full decimal.Decimal) (under, overflow decimal.Decimal, err error)
	AddUsed(ctx context.Context, userID uint64, settleDate time.Time, delta decimal.Decimal) error
	GetUsed(ctx context.Context, userID uint64, settleDate time.Time) (decimal.Decimal, error)
	CreateHold(ctx context.Context, h *CapOverflowHold) error
	ListActiveHolds(ctx context.Context, userID uint64) ([]*CapOverflowHold, error)
	ActiveTotals(ctx context.Context, userID uint64) (usdt, ispay decimal.Decimal, err error)
	ListExpired(ctx context.Context, now time.Time, limit int) ([]*CapOverflowHold, error)
	GetHold(ctx context.Context, id uint64) (*CapOverflowHold, error)
	SaveHold(ctx context.Context, h *CapOverflowHold) error
	StampUnpackaged(ctx context.Context, beforeSettleDate time.Time, clearHours int) error
}

// SplitDailyCap 把一笔动态收益拆成封顶内 / 超额。used 是当日已计入封顶的动态产值。
func SplitDailyCap(full, cap, used decimal.Decimal) (under, overflow decimal.Decimal) {
	full = money.Round(full)
	cap = money.Round(cap)
	used = money.Round(used)
	if !full.IsPositive() {
		return decimal.Zero, decimal.Zero
	}
	if cap.IsNegative() {
		cap = decimal.Zero
	}
	if used.IsNegative() {
		used = decimal.Zero
	}
	remain := money.Round(cap.Sub(used))
	if remain.IsNegative() {
		remain = decimal.Zero
	}
	if !remain.IsPositive() {
		return decimal.Zero, full
	}
	if full.LessThanOrEqual(remain) {
		return full, decimal.Zero
	}
	return remain, money.Round(full.Sub(remain))
}

func isDynamicEntry(entryType string) bool {
	switch entryType {
	case LedgerDirect, LedgerMatch, LedgerManage:
		return true
	default:
		return false
	}
}

func dailyCapDate(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func overflowClearAt(overflowDay time.Time, hours int) time.Time {
	hours = ClampOverflowHours(hours)
	// 次日 0:00 封账，再过 hours 小时到期。
	return dailyCapDate(overflowDay).Add(24 * time.Hour).Add(time.Duration(hours) * time.Hour)
}

// ClampOverflowHours 把清除小时限制在 1–720；非法回落默认 72。
func ClampOverflowHours(hours int) int {
	if hours < minOverflowHours {
		return OverflowClearHours
	}
	if hours > maxOverflowHours {
		return maxOverflowHours
	}
	return hours
}

func holdIsPackaged(h *CapOverflowHold) bool {
	return h != nil && h.ExpiresAt != nil && !h.ExpiresAt.IsZero()
}

func parseRemarkDecimal(remark, key string) (decimal.Decimal, bool) {
	for _, part := range strings.Fields(remark) {
		k, v, ok := strings.Cut(part, "=")
		if !ok || k != key {
			continue
		}
		d, err := decimal.NewFromString(v)
		if err != nil {
			return decimal.Zero, false
		}
		return money.Round(d), true
	}
	return decimal.Zero, false
}

// MatchLedgerCredit 对碰产值（拆分前）。优先 remark credit=，否则 pair×rate，再退回流水 U×2。
func MatchLedgerCredit(m *LedgerEntry) decimal.Decimal {
	if m == nil {
		return decimal.Zero
	}
	if v, ok := parseRemarkDecimal(m.Remark, "credit"); ok && v.IsPositive() {
		return v
	}
	pair, pok := parseRemarkDecimal(m.Remark, "pair")
	rate, rok := parseRemarkDecimal(m.Remark, "rate")
	if pok && rok {
		c := money.Round(pair.Mul(rate))
		if c.IsPositive() {
			return c
		}
	}
	if m.Amount.IsPositive() {
		return money.Round(m.Amount.Mul(decimal.NewFromInt(2)))
	}
	return decimal.Zero
}

type memDailyCap struct {
	mu     sync.Mutex
	used   map[string]decimal.Decimal
	holds  []*CapOverflowHold
	nextID uint64
}

func dailyKey(userID uint64, day time.Time) string {
	return strconv.FormatUint(userID, 10) + "/" + dailyCapDate(day).Format("2006-01-02")
}

func newMemDailyCap() *memDailyCap {
	return &memDailyCap{used: map[string]decimal.Decimal{}, nextID: 1}
}

func (m *memDailyCap) TakeUnder(_ context.Context, userID uint64, settleDate time.Time, cap, full decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := dailyKey(userID, settleDate)
	under, overflow := SplitDailyCap(full, cap, m.used[k])
	if under.IsPositive() {
		m.used[k] = money.Round(m.used[k].Add(under))
	}
	return under, overflow, nil
}

func (m *memDailyCap) AddUsed(_ context.Context, userID uint64, settleDate time.Time, delta decimal.Decimal) error {
	delta = money.Round(delta)
	if !delta.IsPositive() {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	k := dailyKey(userID, settleDate)
	m.used[k] = money.Round(m.used[k].Add(delta))
	return nil
}

func (m *memDailyCap) GetUsed(_ context.Context, userID uint64, settleDate time.Time) (decimal.Decimal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return money.Round(m.used[dailyKey(userID, settleDate)]), nil
}

func (m *memDailyCap) CreateHold(_ context.Context, h *CapOverflowHold) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *h
	cp.ID = m.nextID
	m.nextID++
	m.holds = append(m.holds, &cp)
	return nil
}

func (m *memDailyCap) holdOpen(h *CapOverflowHold) bool {
	return h != nil && h.ReleasedAt == nil && h.BurnedAt == nil && h.Value.IsPositive()
}

func (m *memDailyCap) ListActiveHolds(_ context.Context, userID uint64) ([]*CapOverflowHold, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*CapOverflowHold
	for _, h := range m.holds {
		if h.UserID != userID || !m.holdOpen(h) {
			continue
		}
		cp := *h
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (m *memDailyCap) ActiveTotals(_ context.Context, userID uint64) (decimal.Decimal, decimal.Decimal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var usdt, ispay decimal.Decimal
	for _, h := range m.holds {
		if h.UserID != userID || !m.holdOpen(h) {
			continue
		}
		usdt = usdt.Add(h.USDT)
		ispay = ispay.Add(h.Ispay)
	}
	return money.Round(usdt), money.Round(ispay), nil
}

func (m *memDailyCap) ListExpired(_ context.Context, now time.Time, limit int) ([]*CapOverflowHold, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 {
		limit = 200
	}
	var out []*CapOverflowHold
	for _, h := range m.holds {
		if !m.holdOpen(h) || !holdIsPackaged(h) || h.ExpiresAt.After(now) {
			continue
		}
		cp := *h
		out = append(out, &cp)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (m *memDailyCap) GetHold(_ context.Context, id uint64) (*CapOverflowHold, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, h := range m.holds {
		if h.ID == id {
			cp := *h
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *memDailyCap) SaveHold(_ context.Context, h *CapOverflowHold) error {
	if h == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, row := range m.holds {
		if row.ID == h.ID {
			cp := *h
			m.holds[i] = &cp
			return nil
		}
	}
	return ErrUserNotFound
}

func (m *memDailyCap) StampUnpackaged(_ context.Context, beforeSettleDate time.Time, clearHours int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	before := dailyCapDate(beforeSettleDate)
	hours := ClampOverflowHours(clearHours)
	for _, h := range m.holds {
		if !m.holdOpen(h) || holdIsPackaged(h) {
			continue
		}
		day := dailyCapDate(h.SettleDate)
		if !day.Before(before) {
			continue
		}
		exp := overflowClearAt(day, hours)
		h.ExpiresAt = &exp
	}
	return nil
}
