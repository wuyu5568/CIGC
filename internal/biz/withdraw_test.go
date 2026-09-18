package biz

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

type memWithdraws struct {
	byID map[uint64]*Withdraw
	next uint64
}

func newMemWithdraws() *memWithdraws {
	return &memWithdraws{byID: map[uint64]*Withdraw{}, next: 1}
}

func (m *memWithdraws) Create(_ context.Context, w *Withdraw) (*Withdraw, error) {
	cp := *w
	cp.ID = m.next
	m.next++
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	m.byID[cp.ID] = &cp
	out := cp
	return &out, nil
}

func (m *memWithdraws) FindByID(_ context.Context, id uint64) (*Withdraw, error) {
	w, ok := m.byID[id]
	if !ok {
		return nil, ErrWithdrawNotFound
	}
	cp := *w
	return &cp, nil
}

func (m *memWithdraws) CasStatus(_ context.Context, id uint64, from, to, remark string, reviewedAt *time.Time) error {
	w, ok := m.byID[id]
	if !ok {
		return ErrWithdrawNotFound
	}
	if w.Status != from {
		return ErrWithdrawConflict
	}
	w.Status = to
	w.Remark = remark
	w.ReviewedAt = reviewedAt
	return nil
}

func (m *memWithdraws) ListByUser(_ context.Context, userID uint64) ([]*Withdraw, error) {
	var out []*Withdraw
	for _, w := range m.byID {
		if w.UserID == userID {
			cp := *w
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (m *memWithdraws) ListAdmin(_ context.Context, address, status, asset string, page, pageSize int) ([]*AdminWithdrawRow, int, error) {
	_ = address
	var filtered []*AdminWithdrawRow
	for _, w := range m.byID {
		if status != "" && w.Status != status {
			continue
		}
		if asset != "" && withdrawAssetOf(w) != asset {
			continue
		}
		cp := *w
		filtered = append(filtered, &AdminWithdrawRow{Withdraw: cp})
	}
	total := len(filtered)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultRewardPageSize
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []*AdminWithdrawRow{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return filtered[start:end], total, nil
}

func (m *memWithdraws) ListPayoutQueue(_ context.Context, limit int) ([]*AdminWithdrawRow, error) {
	if limit < 1 {
		limit = 20
	}
	var out []*AdminWithdrawRow
	for _, w := range m.byID {
		if withdrawAssetOf(w) != WithdrawAssetUSDT && withdrawAssetOf(w) != WithdrawAssetIspay {
			continue
		}
		if w.Status != WithdrawRewarded && w.Status != WithdrawDoing {
			continue
		}
		cp := *w
		out = append(out, &AdminWithdrawRow{Withdraw: cp})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *memWithdraws) SumUsedToday(_ context.Context, userID uint64, asset string, from, to time.Time) (decimal.Decimal, error) {
	sum := decimal.Zero
	for _, w := range m.byID {
		if w.UserID != userID || !withdrawCountsDaily(w.Status) {
			continue
		}
		if withdrawAssetOf(w) != asset {
			continue
		}
		if w.CreatedAt.Before(from) || !w.CreatedAt.Before(to) {
			continue
		}
		sum = sum.Add(w.Amount)
	}
	return sum, nil
}

func (m *memWithdraws) UpdatePayoutMeta(_ context.Context, id uint64, txHash, payoutError string) error {
	w, ok := m.byID[id]
	if !ok {
		return ErrWithdrawNotFound
	}
	w.TxHash = txHash
	w.PayoutError = payoutError
	return nil
}

type memConfigs struct {
	min             string
	minIspay        string
	feeRate         string
	feeRateIspay    string
	dailyLimit      string
	dailyLimitIspay string
	directRate      string
	matchRate       string
	manageRate      string
	manageGens      string
	ispayPrice      string
	overflowHours   string
	withdrawEnabled string
	payoutMaxIspay  string
	rows            []*BusinessConfig
}

func (m *memConfigs) GetValue(_ context.Context, key string) (string, error) {
	switch key {
	case ConfigMinWithdraw:
		if m.min != "" {
			return m.min, nil
		}
	case ConfigMinWithdrawIspay:
		if m.minIspay != "" {
			return m.minIspay, nil
		}
	case ConfigWithdrawFeeRate:
		if m.feeRate != "" {
			return m.feeRate, nil
		}
	case ConfigWithdrawFeeIspay:
		if m.feeRateIspay != "" {
			return m.feeRateIspay, nil
		}
	case ConfigWithdrawDaily:
		if m.dailyLimit != "" {
			return m.dailyLimit, nil
		}
	case ConfigWithdrawDailyIspay:
		if m.dailyLimitIspay != "" {
			return m.dailyLimitIspay, nil
		}
	case ConfigDirectRate:
		if m.directRate != "" {
			return m.directRate, nil
		}
	case ConfigMatchRate:
		if m.matchRate != "" {
			return m.matchRate, nil
		}
	case ConfigManageRate:
		if m.manageRate != "" {
			return m.manageRate, nil
		}
	case ConfigManageGens:
		if m.manageGens != "" {
			return m.manageGens, nil
		}
	case ConfigIspayPrice:
		if m.ispayPrice != "" {
			return m.ispayPrice, nil
		}
	case ConfigOverflowHours:
		if m.overflowHours != "" {
			return m.overflowHours, nil
		}
	case ConfigWithdrawEnabled:
		if m.withdrawEnabled != "" {
			return m.withdrawEnabled, nil
		}
	case ConfigPayoutMaxIspay:
		if m.payoutMaxIspay != "" {
			return m.payoutMaxIspay, nil
		}
	}
	for _, r := range m.rows {
		if r.Key == key {
			return r.Value, nil
		}
	}
	return "", nil
}

func (m *memConfigs) List(_ context.Context) ([]*BusinessConfig, error) {
	out := make([]*BusinessConfig, 0, len(m.rows))
	for _, r := range m.rows {
		cp := *r
		out = append(out, &cp)
	}
	return out, nil
}

func (m *memConfigs) FindByID(_ context.Context, id uint64) (*BusinessConfig, error) {
	for _, r := range m.rows {
		if r.ID == id {
			cp := *r
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *memConfigs) SetValue(_ context.Context, id uint64, value string) error {
	for _, r := range m.rows {
		if r.ID == id {
			r.Value = value
			switch r.Key {
			case ConfigMinWithdraw:
				m.min = value
			case ConfigMinWithdrawIspay:
				m.minIspay = value
			case ConfigWithdrawFeeRate:
				m.feeRate = value
			case ConfigWithdrawFeeIspay:
				m.feeRateIspay = value
			case ConfigDirectRate:
				m.directRate = value
			case ConfigMatchRate:
				m.matchRate = value
			case ConfigManageRate:
				m.manageRate = value
			case ConfigManageGens:
				m.manageGens = value
			case ConfigIspayPrice:
				m.ispayPrice = value
			case ConfigOverflowHours:
				m.overflowHours = value
			case ConfigWithdrawEnabled:
				m.withdrawEnabled = value
			case ConfigPayoutMaxIspay:
				m.payoutMaxIspay = value
			}
			return nil
		}
	}
	return ErrConfigNotFound
}

func (m *memConfigs) Upsert(_ context.Context, row *BusinessConfig) error {
	if row == nil || row.Key == "" {
		return ErrConfigInvalid
	}
	for _, r := range m.rows {
		if r.Key == row.Key {
			r.Name = row.Name
			r.Value = row.Value
			r.SortOrder = row.SortOrder
			return nil
		}
	}
	cp := *row
	if cp.ID == 0 {
		cp.ID = uint64(len(m.rows) + 1)
	}
	m.rows = append(m.rows, &cp)
	return nil
}

func newWithdrawUC(users *memUsers, led *memLedger, wds *memWithdraws, min string) *WithdrawUseCase {
	return NewWithdrawUseCase(users, users, led, wds, &memConfigs{min: min}, NopTx{})
}

func TestCreateWithdraw_FreezesAvailable(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("100"),
		PaidAmount:       decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	led := &memLedger{}
	wds := newMemWithdraws()
	uc := newWithdrawUC(users, led, wds, "10")

	wd, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("20"))
	if err != nil {
		t.Fatal(err)
	}
	if wd.Status != WithdrawRewarded || !wd.FeeAmount.Equal(decimal.RequireFromString("2")) || !wd.CreditedAmount.Equal(decimal.RequireFromString("18")) {
		t.Fatalf("%+v", wd)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.AvailableBalance.Equal(decimal.RequireFromString("80")) || !got.FrozenBalance.Equal(decimal.RequireFromString("20")) {
		t.Fatalf("avail=%s frozen=%s", got.AvailableBalance, got.FrozenBalance)
	}
	if len(led.rows) != 2 {
		t.Fatalf("ledger n=%d", len(led.rows))
	}
}

func TestCreateWithdraw_FeeRateOneRejected(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("100"),
		PaidAmount:       decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewWithdrawUseCase(users, users, &memLedger{}, newMemWithdraws(), &memConfigs{min: "10", feeRate: "1"}, NopTx{})
	if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("10")); !errors.Is(err, ErrWithdrawFeeExceeds) {
		t.Fatalf("got %v", err)
	}
}

func TestCancelWithdraw_UnfreezesOwnPending(t *testing.T) {
	users := newMemUsers()
	owner, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("50"),
		PaidAmount:       decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	other, err := users.Create(context.Background(), &User{
		Address:          "0xdef",
		AvailableBalance: decimal.RequireFromString("50"),
		PaidAmount:       decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	led := &memLedger{}
	uc := newWithdrawUC(users, led, newMemWithdraws(), "10")
	wd, err := uc.Create(context.Background(), owner.ID, decimal.RequireFromString("10"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Cancel(context.Background(), other.ID, wd.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("other: %v", err)
	}
	got, err := uc.Cancel(context.Background(), owner.ID, wd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != WithdrawCancelled {
		t.Fatalf("status %s", got.Status)
	}
	u2, err := users.FindByID(context.Background(), owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !u2.AvailableBalance.Equal(decimal.RequireFromString("50")) || !u2.FrozenBalance.IsZero() {
		t.Fatalf("avail=%s frozen=%s", u2.AvailableBalance, u2.FrozenBalance)
	}
	if _, err := uc.Cancel(context.Background(), owner.ID, wd.ID); !errors.Is(err, ErrWithdrawConflict) {
		t.Fatalf("double: %v", err)
	}
}

func TestCreateWithdraw_PerTxCap(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("100"),
		PaidAmount:       decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewWithdrawUseCase(users, users, &memLedger{}, newMemWithdraws(), &memConfigs{min: "10", feeRate: "0", dailyLimit: "20"}, NopTx{})
	if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("10")); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("10")); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("20")); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("21")); !errors.Is(err, ErrWithdrawDailyCap) {
		t.Fatalf("over per-tx: %v", err)
	}
}

func TestCreateWithdraw_IspayPerTxCapSeparate(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("100"),
		IspayBalance:     decimal.RequireFromString("10"),
		PaidAmount:       decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewWithdrawUseCase(users, users, &memLedger{}, newMemWithdraws(), &memConfigs{
		min: "10", feeRate: "0", dailyLimit: "1000", dailyLimitIspay: "2",
	}, NopTx{})
	if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("50")); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.CreateAsset(context.Background(), u.ID, decimal.RequireFromString("2"), WithdrawAssetIspay); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.CreateAsset(context.Background(), u.ID, decimal.RequireFromString("1"), WithdrawAssetIspay); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.CreateAsset(context.Background(), u.ID, decimal.RequireFromString("3"), WithdrawAssetIspay); !errors.Is(err, ErrWithdrawDailyCap) {
		t.Fatalf("ispay over per-tx: %v", err)
	}
	lim := uc.UserLimits(context.Background(), u.ID)
	if !lim.DailyTwo.Equal(decimal.RequireFromString("2")) || !lim.RemainTwo.Equal(decimal.RequireFromString("2")) {
		t.Fatalf("ispay cap=%s remain=%s", lim.DailyTwo, lim.RemainTwo)
	}
}

func TestCreateWithdraw_IspayMinAndFee(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:      "0xabc",
		IspayBalance: decimal.RequireFromString("20"),
		PaidAmount:   decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewWithdrawUseCase(users, users, &memLedger{}, newMemWithdraws(), &memConfigs{
		min: "10", minIspay: "5", feeRateIspay: "0.10",
	}, NopTx{})
	lim := uc.UserLimits(context.Background(), u.ID)
	if !lim.MinTwo.Equal(decimal.RequireFromString("5")) || !lim.RateTwo.Equal(decimal.RequireFromString("0.10")) {
		t.Fatalf("limits min=%s rate=%s", lim.MinTwo, lim.RateTwo)
	}
	if _, err := uc.CreateAsset(context.Background(), u.ID, decimal.RequireFromString("4"), WithdrawAssetIspay); !errors.Is(err, ErrWithdrawBelowMin) {
		t.Fatalf("below min: %v", err)
	}
	wd, err := uc.CreateAsset(context.Background(), u.ID, decimal.RequireFromString("10"), WithdrawAssetIspay)
	if err != nil {
		t.Fatal(err)
	}
	if !wd.FeeAmount.Equal(decimal.RequireFromString("1")) || !wd.CreditedAmount.Equal(decimal.RequireFromString("9")) {
		t.Fatalf("fee=%s credited=%s", wd.FeeAmount, wd.CreditedAmount)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.IspayBalance.Equal(decimal.RequireFromString("10")) || !got.FrozenIspay.Equal(decimal.RequireFromString("10")) {
		t.Fatalf("freeze application amount, ispay=%s frozen=%s", got.IspayBalance, got.FrozenIspay)
	}
}

func TestCreateWithdraw_ClosedByConfig(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("100"),
		PaidAmount:       decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewWithdrawUseCase(users, users, &memLedger{}, newMemWithdraws(), &memConfigs{min: "10", withdrawEnabled: "0"}, NopTx{})
	if uc.Enabled(context.Background()) {
		t.Fatal("should be closed")
	}
	if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("20")); !errors.Is(err, ErrWithdrawClosed) {
		t.Fatalf("closed: %v", err)
	}
}

func TestCreateWithdraw_DailyCapZeroUnlimited(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("80"),
		PaidAmount:       decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewWithdrawUseCase(users, users, &memLedger{}, newMemWithdraws(), &memConfigs{min: "10", feeRate: "0", dailyLimit: "0"}, NopTx{})
	if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("40")); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("40")); err != nil {
		t.Fatal(err)
	}
}

func TestCreateWithdraw_BelowMinAndInsufficient(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("15"),
		PaidAmount:       decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := newWithdrawUC(users, &memLedger{}, newMemWithdraws(), "10")
	if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("9")); !errors.Is(err, ErrWithdrawBelowMin) {
		t.Fatalf("min: %v", err)
	}
	if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("16")); !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("bal: %v", err)
	}
}

func TestCreateWithdraw_DisabledUser(t *testing.T) {
	users := newMemUsers()
	now := time.Now()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("100"),
		DisabledAt:       &now,
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := newWithdrawUC(users, &memLedger{}, newMemWithdraws(), "10")
	if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("10")); !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("got %v", err)
	}
}

func TestCreateWithdraw_InactiveUser(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("100"),
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := newWithdrawUC(users, &memLedger{}, newMemWithdraws(), "10")
	if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("10")); !errors.Is(err, ErrUserInactive) {
		t.Fatalf("got %v", err)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.AvailableBalance.Equal(decimal.RequireFromString("100")) || !got.FrozenBalance.IsZero() {
		t.Fatalf("should not freeze inactive avail=%s frozen=%s", got.AvailableBalance, got.FrozenBalance)
	}
}

func TestCreateWithdraw_IspayInactiveAndFreeze(t *testing.T) {
	users := newMemUsers()
	inactive, err := users.Create(context.Background(), &User{
		Address:      "0xabc",
		IspayBalance: decimal.RequireFromString("5"),
	})
	if err != nil {
		t.Fatal(err)
	}
	active, err := users.Create(context.Background(), &User{
		Address:          "0xdef",
		AvailableBalance: decimal.RequireFromString("50"),
		IspayBalance:     decimal.RequireFromString("5"),
		PaidAmount:       decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := newWithdrawUC(users, &memLedger{}, newMemWithdraws(), "10")
	if _, err := uc.CreateAsset(context.Background(), inactive.ID, decimal.RequireFromString("1"), WithdrawAssetIspay); !errors.Is(err, ErrUserInactive) {
		t.Fatalf("inactive: %v", err)
	}
	wd, err := uc.CreateAsset(context.Background(), active.ID, decimal.RequireFromString("1"), WithdrawAssetIspay)
	if err != nil {
		t.Fatal(err)
	}
	if wd.Asset != WithdrawAssetIspay || wd.Status != WithdrawRewarded {
		t.Fatalf("asset=%s status=%s", wd.Asset, wd.Status)
	}
	got, err := users.FindByID(context.Background(), active.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.IspayBalance.Equal(decimal.RequireFromString("4")) || !got.FrozenIspay.Equal(decimal.RequireFromString("1")) {
		t.Fatalf("ispay=%s frozen_ispay=%s", got.IspayBalance, got.FrozenIspay)
	}
	if !got.AvailableBalance.Equal(decimal.RequireFromString("50")) || !got.FrozenBalance.IsZero() {
		t.Fatalf("must not touch usdt avail=%s frozen=%s", got.AvailableBalance, got.FrozenBalance)
	}
	if _, err := uc.CreateAsset(context.Background(), active.ID, decimal.RequireFromString("6"), WithdrawAssetIspay); !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("over: %v", err)
	}
	if asset, err := ParseWithdrawAsset("3"); err != nil || asset != WithdrawAssetIspay {
		t.Fatalf("parse 3: %s %v", asset, err)
	}
	if _, err := ParseWithdrawAsset("btc"); !errors.Is(err, ErrInvalidWithdrawAsset) {
		t.Fatalf("parse btc: %v", err)
	}
}

func TestWithdrawIspayPassAndReject(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:      "0xabc",
		IspayBalance: decimal.RequireFromString("5"),
		PaidAmount:   decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	wds := newMemWithdraws()
	uc := newWithdrawUC(users, &memLedger{}, wds, "10")
	passWD, err := uc.CreateAsset(context.Background(), u.ID, decimal.RequireFromString("1"), WithdrawAssetIspay)
	if err != nil {
		t.Fatal(err)
	}
	if passWD.Status != WithdrawRewarded {
		t.Fatalf("new ispay status %s", passWD.Status)
	}
	if _, err := uc.Pass(context.Background(), passWD.ID); !errors.Is(err, ErrWithdrawConflict) {
		t.Fatalf("rewarded pass: %v", err)
	}
	legacy, err := wds.Create(context.Background(), &Withdraw{
		UserID:         u.ID,
		Amount:         decimal.RequireFromString("0.5"),
		FeeAmount:      decimal.Zero,
		CreditedAmount: decimal.RequireFromString("0.5"),
		Asset:          WithdrawAssetIspay,
		Status:         WithdrawPending,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Pass(context.Background(), legacy.ID); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.FrozenIspay.Equal(decimal.RequireFromString("1")) || !got.IspayBalance.Equal(decimal.RequireFromString("4")) {
		t.Fatalf("pass ispay=%s frozen=%s", got.IspayBalance, got.FrozenIspay)
	}

	rejectWD, err := uc.CreateAsset(context.Background(), u.ID, decimal.RequireFromString("2"), WithdrawAssetIspay)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Reject(context.Background(), rejectWD.ID); err != nil {
		t.Fatal(err)
	}
	got, err = users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.IspayBalance.Equal(decimal.RequireFromString("4")) || !got.FrozenIspay.Equal(decimal.RequireFromString("1")) {
		t.Fatalf("reject ispay=%s frozen=%s", got.IspayBalance, got.FrozenIspay)
	}
}

func TestWithdrawPassKeepsFrozen(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("50"),
		PaidAmount:       decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := newWithdrawUC(users, &memLedger{}, newMemWithdraws(), "10")
	wd, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("10"))
	if err != nil {
		t.Fatal(err)
	}
	if wd.Status != WithdrawRewarded {
		t.Fatalf("status %s", wd.Status)
	}
	u2, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !u2.FrozenBalance.Equal(decimal.RequireFromString("10")) {
		t.Fatalf("frozen=%s", u2.FrozenBalance)
	}
	if _, err := uc.Pass(context.Background(), wd.ID); !errors.Is(err, ErrWithdrawConflict) {
		t.Fatalf("usdt pass: %v", err)
	}
}

func TestWithdrawRejectUnfreezes(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("50"),
		PaidAmount:       decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	led := &memLedger{}
	uc := newWithdrawUC(users, led, newMemWithdraws(), "10")
	wd, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("10"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := uc.Reject(context.Background(), wd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != WithdrawRejected {
		t.Fatalf("status %s", got.Status)
	}
	u2, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !u2.AvailableBalance.Equal(decimal.RequireFromString("50")) || !u2.FrozenBalance.IsZero() {
		t.Fatalf("avail=%s frozen=%s", u2.AvailableBalance, u2.FrozenBalance)
	}
	unfreeze := 0
	for _, e := range led.rows {
		if e.EntryType == LedgerUnfreeze {
			unfreeze++
		}
	}
	if unfreeze != 2 {
		t.Fatalf("unfreeze entries=%d", unfreeze)
	}
}

func TestListUserWithdrawPagination(t *testing.T) {
	users := newMemUsers()
	u, err := users.Create(context.Background(), &User{
		Address:          "0xabc",
		AvailableBalance: decimal.RequireFromString("200"),
		PaidAmount:       decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := newWithdrawUC(users, &memLedger{}, newMemWithdraws(), "10")
	for i := 0; i < 11; i++ {
		if _, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("10")); err != nil {
			t.Fatal(err)
		}
	}
	p1, err := uc.ListUser(context.Background(), u.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if p1.Total != 11 || len(p1.Items) != 10 {
		t.Fatalf("p1 total=%d n=%d", p1.Total, len(p1.Items))
	}
}

type memPayer struct {
	hash    string
	from    string
	sendErr error
	ok      bool
	pending bool
	recErr  error
	sends   int
}

func (m *memPayer) FromAddress() string { return m.from }

func (m *memPayer) Transfer(_ context.Context, _ string, _ string, _ decimal.Decimal) (string, error) {
	m.sends++
	return m.hash, m.sendErr
}

func (m *memPayer) Receipt(_ context.Context, _ string) (bool, bool, error) {
	return m.ok, m.pending, m.recErr
}

func rewardedUSDT(t *testing.T, users *memUsers, uc *WithdrawUseCase, avail string) (*User, *Withdraw) {
	t.Helper()
	u, err := users.Create(context.Background(), &User{
		Address:          "0x1111111111111111111111111111111111111111",
		AvailableBalance: decimal.RequireFromString(avail),
		PaidAmount:       decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	wd, err := uc.Create(context.Background(), u.ID, decimal.RequireFromString("10"))
	if err != nil {
		t.Fatal(err)
	}
	return u, wd
}

func TestRunPayout_Disabled(t *testing.T) {
	uc := newWithdrawUC(newMemUsers(), &memLedger{}, newMemWithdraws(), "10")
	if _, err := uc.RunPayout(context.Background(), 0); !errors.Is(err, ErrPayoutDisabled) {
		t.Fatalf("got %v", err)
	}
}

func TestRunPayout_SuccessBurnsFrozen(t *testing.T) {
	users := newMemUsers()
	uc := newWithdrawUC(users, &memLedger{}, newMemWithdraws(), "10")
	p := &memPayer{hash: "0xabc", ok: true, from: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	uc.SetPayout(p, true, decimal.RequireFromString("100"))
	if uc.HotWalletAddress() != p.from {
		t.Fatalf("hot=%s", uc.HotWalletAddress())
	}
	u, _ := rewardedUSDT(t, users, uc, "50")
	res, err := uc.RunPayout(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Passed != 1 || res.Sent != 1 || p.sends != 1 {
		t.Fatalf("%+v sends=%d", res, p.sends)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.FrozenBalance.IsZero() || !got.AvailableBalance.Equal(decimal.RequireFromString("40")) {
		t.Fatalf("avail=%s frozen=%s", got.AvailableBalance, got.FrozenBalance)
	}
	wd, err := uc.withdraws.FindByID(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if wd.Status != WithdrawPass || wd.TxHash != "0xabc" {
		t.Fatalf("%+v", wd)
	}
}

func TestRunPayout_SendFailKeepsFrozen(t *testing.T) {
	users := newMemUsers()
	uc := newWithdrawUC(users, &memLedger{}, newMemWithdraws(), "10")
	uc.SetPayout(&memPayer{sendErr: errors.New("rpc down")}, true, decimal.RequireFromString("100"))
	u, wd := rewardedUSDT(t, users, uc, "50")
	res, err := uc.RunPayout(context.Background(), wd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("%+v", res)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.FrozenBalance.Equal(decimal.RequireFromString("10")) {
		t.Fatalf("frozen=%s", got.FrozenBalance)
	}
	wd, err = uc.withdraws.FindByID(context.Background(), wd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if wd.Status != WithdrawRewarded {
		t.Fatalf("status %s", wd.Status)
	}
}

func TestRunPayout_SkipOverMaxAndIspay(t *testing.T) {
	users := newMemUsers()
	uc := newWithdrawUC(users, &memLedger{}, newMemWithdraws(), "10")
	p := &memPayer{hash: "0x1", ok: true}
	uc.SetPayout(p, true, decimal.RequireFromString("5"))
	_, wd := rewardedUSDT(t, users, uc, "50")
	res, err := uc.RunPayout(context.Background(), wd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped != 1 || p.sends != 0 {
		t.Fatalf("%+v sends=%d", res, p.sends)
	}
}

func TestRunPayout_IspayPaysAndUnfreezes(t *testing.T) {
	users := newMemUsers()
	cfg := &memConfigs{min: "10", payoutMaxIspay: "100"}
	uc := NewWithdrawUseCase(users, users, &memLedger{}, newMemWithdraws(), cfg, NopTx{})
	p := &memPayer{hash: "0xispay", ok: true}
	uc.SetPayout(p, true, decimal.RequireFromString("100"))
	u, err := users.Create(context.Background(), &User{
		Address:      "0x2222222222222222222222222222222222222222",
		IspayBalance: decimal.RequireFromString("5"),
		PaidAmount:   decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	wd, err := uc.CreateAsset(context.Background(), u.ID, decimal.RequireFromString("1"), WithdrawAssetIspay)
	if err != nil {
		t.Fatal(err)
	}
	if wd.Status != WithdrawRewarded {
		t.Fatalf("status %s", wd.Status)
	}
	res, err := uc.RunPayout(context.Background(), wd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Sent != 1 || res.Passed != 1 || p.sends != 1 {
		t.Fatalf("%+v sends=%d", res, p.sends)
	}
	got, err := users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.FrozenIspay.IsZero() || !got.IspayBalance.Equal(decimal.RequireFromString("4")) {
		t.Fatalf("ispay=%s frozen=%s", got.IspayBalance, got.FrozenIspay)
	}
	wd, err = uc.withdraws.FindByID(context.Background(), wd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if wd.Status != WithdrawPass || wd.TxHash != "0xispay" {
		t.Fatalf("%+v", wd)
	}
}

func TestRunPayout_SkipIspayOverAdminMax(t *testing.T) {
	users := newMemUsers()
	cfg := &memConfigs{min: "10", payoutMaxIspay: "1"}
	uc := NewWithdrawUseCase(users, users, &memLedger{}, newMemWithdraws(), cfg, NopTx{})
	p := &memPayer{hash: "0x1", ok: true}
	uc.SetPayout(p, true, decimal.RequireFromString("100"))
	u, err := users.Create(context.Background(), &User{
		Address:      "0x2222222222222222222222222222222222222222",
		IspayBalance: decimal.RequireFromString("5"),
		PaidAmount:   decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	wd, err := uc.CreateAsset(context.Background(), u.ID, decimal.RequireFromString("2"), WithdrawAssetIspay)
	if err != nil {
		t.Fatal(err)
	}
	res, err := uc.RunPayout(context.Background(), wd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped != 1 || p.sends != 0 {
		t.Fatalf("%+v sends=%d", res, p.sends)
	}
}

func TestRunPayout_PendingReceiptThenPass(t *testing.T) {
	users := newMemUsers()
	uc := newWithdrawUC(users, &memLedger{}, newMemWithdraws(), "10")
	p := &memPayer{hash: "0xpend", pending: true}
	uc.SetPayout(p, true, decimal.RequireFromString("100"))
	_, wd := rewardedUSDT(t, users, uc, "50")
	res, err := uc.RunPayout(context.Background(), wd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Sent != 1 || res.Passed != 0 {
		t.Fatalf("%+v", res)
	}
	got, err := uc.withdraws.FindByID(context.Background(), wd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != WithdrawDoing || got.TxHash != "0xpend" {
		t.Fatalf("%+v", got)
	}
	p.pending = false
	p.ok = true
	res, err = uc.RunPayout(context.Background(), wd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Passed != 1 || p.sends != 1 {
		t.Fatalf("second %+v sends=%d", res, p.sends)
	}
}
