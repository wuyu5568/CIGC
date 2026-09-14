package biz

import (
	"context"
	"testing"
	"time"

	"github.com/cigc/app/internal/conf"
	"github.com/shopspring/decimal"
)

type stubVerifier struct{ err error }

func (s stubVerifier) Verify(_, _, _ string) error { return s.err }

type stubTokens struct{}

func (stubTokens) Issue(userID uint64, address string) (string, error) {
	return "tok-" + address, nil
}
func (stubTokens) IssueAdmin() (string, error) { return "admin-tok", nil }

type memUsers struct {
	byID      map[uint64]*User
	byAddress map[string]*User
	next      uint64
}

func newMemUsers() *memUsers {
	return &memUsers{byID: map[uint64]*User{}, byAddress: map[string]*User{}, next: 1}
}

func (m *memUsers) FindByAddress(_ context.Context, address string) (*User, error) {
	u, ok := m.byAddress[address]
	if !ok {
		return nil, ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *memUsers) FindByID(_ context.Context, id uint64) (*User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *memUsers) Create(_ context.Context, u *User) (*User, error) {
	cp := *u
	cp.ID = m.next
	m.next++
	m.byID[cp.ID] = &cp
	m.byAddress[cp.Address] = &cp
	out := cp
	return &out, nil
}

func (m *memUsers) AddPaidAmount(_ context.Context, userID uint64, delta decimal.Decimal) error {
	u, ok := m.byID[userID]
	if !ok {
		return ErrUserNotFound
	}
	u.PaidAmount = u.PaidAmount.Add(delta)
	if other, ok := m.byAddress[u.Address]; ok {
		other.PaidAmount = u.PaidAmount
	}
	return nil
}

func (m *memUsers) ListAll(_ context.Context) ([]*User, error) {
	out := make([]*User, 0, len(m.byID))
	for _, u := range m.byID {
		cp := *u
		out = append(out, &cp)
	}
	return out, nil
}

func (m *memUsers) SetCapEffective(_ context.Context, userID uint64, cap decimal.Decimal) error {
	u, ok := m.byID[userID]
	if !ok {
		return ErrUserNotFound
	}
	u.CapEffective = cap
	if other, ok := m.byAddress[u.Address]; ok {
		other.CapEffective = cap
	}
	return nil
}

func (m *memUsers) ListAdmin(_ context.Context, address string, page, pageSize int) ([]*User, int, error) {
	var filtered []*User
	for _, u := range m.byID {
		if address != "" && !containsFold(u.Address, address) {
			continue
		}
		cp := *u
		filtered = append(filtered, &cp)
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
		return []*User{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return filtered[start:end], total, nil
}

func (m *memUsers) ListByInviter(_ context.Context, inviterID uint64) ([]*User, error) {
	var out []*User
	for _, u := range m.byID {
		if u.InviterID != nil && *u.InviterID == inviterID {
			cp := *u
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (m *memUsers) SetDisabledAt(_ context.Context, userID uint64, disabledAt *time.Time) error {
	u, ok := m.byID[userID]
	if !ok {
		return ErrUserNotFound
	}
	u.DisabledAt = disabledAt
	if other, ok := m.byAddress[u.Address]; ok {
		other.DisabledAt = disabledAt
	}
	return nil
}

func (m *memUsers) AddAvailableBalance(_ context.Context, userID uint64, delta decimal.Decimal) error {
	u, ok := m.byID[userID]
	if !ok {
		return ErrUserNotFound
	}
	u.AvailableBalance = u.AvailableBalance.Add(delta)
	return nil
}

func (m *memUsers) SubAvailableBalance(_ context.Context, userID uint64, delta decimal.Decimal) error {
	u, ok := m.byID[userID]
	if !ok {
		return ErrUserNotFound
	}
	if u.AvailableBalance.LessThan(delta) {
		return ErrInsufficientBalance
	}
	u.AvailableBalance = u.AvailableBalance.Sub(delta)
	return nil
}

func (m *memUsers) AddRechargeBalance(_ context.Context, userID uint64, delta decimal.Decimal) error {
	u, ok := m.byID[userID]
	if !ok {
		return ErrUserNotFound
	}
	u.RechargeBalance = u.RechargeBalance.Add(delta)
	return nil
}

func (m *memUsers) SubRechargeBalance(_ context.Context, userID uint64, delta decimal.Decimal) error {
	u, ok := m.byID[userID]
	if !ok {
		return ErrUserNotFound
	}
	if u.RechargeBalance.LessThan(delta) {
		return ErrInsufficientBalance
	}
	u.RechargeBalance = u.RechargeBalance.Sub(delta)
	return nil
}

func (m *memUsers) AddIspayBalance(_ context.Context, userID uint64, delta decimal.Decimal) error {
	u, ok := m.byID[userID]
	if !ok {
		return ErrUserNotFound
	}
	u.IspayBalance = u.IspayBalance.Add(delta)
	return nil
}

func (m *memUsers) SubIspayBalance(_ context.Context, userID uint64, delta decimal.Decimal) error {
	u, ok := m.byID[userID]
	if !ok {
		return ErrUserNotFound
	}
	if u.IspayBalance.LessThan(delta) {
		return ErrInsufficientBalance
	}
	u.IspayBalance = u.IspayBalance.Sub(delta)
	return nil
}

func (m *memUsers) AddLockBalance(_ context.Context, userID uint64, delta decimal.Decimal) error {
	u, ok := m.byID[userID]
	if !ok {
		return ErrUserNotFound
	}
	u.LockBalance = u.LockBalance.Add(delta)
	return nil
}

func (m *memUsers) SubLockBalance(_ context.Context, userID uint64, delta decimal.Decimal) error {
	u, ok := m.byID[userID]
	if !ok {
		return ErrUserNotFound
	}
	if u.LockBalance.LessThan(delta) {
		return ErrInsufficientBalance
	}
	u.LockBalance = u.LockBalance.Sub(delta)
	return nil
}

func (m *memUsers) AddLockIspay(_ context.Context, userID uint64, delta decimal.Decimal) error {
	u, ok := m.byID[userID]
	if !ok {
		return ErrUserNotFound
	}
	u.LockIspay = u.LockIspay.Add(delta)
	return nil
}

func (m *memUsers) SubLockIspay(_ context.Context, userID uint64, delta decimal.Decimal) error {
	u, ok := m.byID[userID]
	if !ok {
		return ErrUserNotFound
	}
	if u.LockIspay.LessThan(delta) {
		return ErrInsufficientBalance
	}
	u.LockIspay = u.LockIspay.Sub(delta)
	return nil
}

func (m *memUsers) AddFrozenBalance(_ context.Context, userID uint64, delta decimal.Decimal) error {
	u, ok := m.byID[userID]
	if !ok {
		return ErrUserNotFound
	}
	u.FrozenBalance = u.FrozenBalance.Add(delta)
	return nil
}

func (m *memUsers) SubFrozenBalance(_ context.Context, userID uint64, delta decimal.Decimal) error {
	u, ok := m.byID[userID]
	if !ok {
		return ErrUserNotFound
	}
	if u.FrozenBalance.LessThan(delta) {
		return ErrInsufficientBalance
	}
	u.FrozenBalance = u.FrozenBalance.Sub(delta)
	return nil
}

func (m *memUsers) AddFrozenIspay(_ context.Context, userID uint64, delta decimal.Decimal) error {
	u, ok := m.byID[userID]
	if !ok {
		return ErrUserNotFound
	}
	u.FrozenIspay = u.FrozenIspay.Add(delta)
	return nil
}

func (m *memUsers) SubFrozenIspay(_ context.Context, userID uint64, delta decimal.Decimal) error {
	u, ok := m.byID[userID]
	if !ok {
		return ErrUserNotFound
	}
	if u.FrozenIspay.LessThan(delta) {
		return ErrInsufficientBalance
	}
	u.FrozenIspay = u.FrozenIspay.Sub(delta)
	return nil
}

func containsFold(s, sub string) bool {
	if sub == "" {
		return true
	}
	return len(s) >= len(sub) && (s == sub || stringContains(s, sub))
}

func stringContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

type memRecommends struct {
	path map[uint64]string
}

func (m *memRecommends) GetPath(_ context.Context, userID uint64) (string, error) {
	return m.path[userID], nil
}

func (m *memRecommends) SavePath(_ context.Context, userID uint64, path string) error {
	if m.path == nil {
		m.path = map[uint64]string{}
	}
	m.path[userID] = path
	return nil
}

const (
	addrGenesis = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	addrUserB   = "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	addrUserC   = "0xcccccccccccccccccccccccccccccccccccccccc"
)

func TestEthAuthorize_GenesisNoInvite(t *testing.T) {
	users := newMemUsers()
	uc := NewUserUseCase(users, &memRecommends{path: map[uint64]string{}}, stubVerifier{}, stubTokens{}, &conf.Auth{}, addrGenesis, nil)
	res, err := uc.EthAuthorize(context.Background(), addrGenesis, "sig", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.User.ID == 0 {
		t.Fatal("expected created user")
	}
}

func TestEthAuthorize_InviteRequired(t *testing.T) {
	users := newMemUsers()
	uc := NewUserUseCase(users, &memRecommends{path: map[uint64]string{}}, stubVerifier{}, stubTokens{}, &conf.Auth{}, addrGenesis, nil)
	_, err := uc.EthAuthorize(context.Background(), addrUserB, "sig", "")
	if err != ErrInviteRequired {
		t.Fatalf("got %v", err)
	}
}

func TestEthAuthorize_InvalidInvite(t *testing.T) {
	users := newMemUsers()
	uc := NewUserUseCase(users, &memRecommends{path: map[uint64]string{}}, stubVerifier{}, stubTokens{}, &conf.Auth{}, addrGenesis, nil)
	if _, err := uc.EthAuthorize(context.Background(), addrGenesis, "sig", ""); err != nil {
		t.Fatal(err)
	}
	_, err := uc.EthAuthorize(context.Background(), addrUserB, "sig", addrUserC)
	if err != ErrInviteInvalid {
		t.Fatalf("got %v", err)
	}
}

func TestAdminLogin(t *testing.T) {
	uc := NewUserUseCase(newMemUsers(), &memRecommends{}, stubVerifier{}, stubTokens{}, &conf.Auth{
		AdminUsername: "admin",
		AdminPassword: "pw",
	}, "", nil)
	tok, err := uc.AdminLogin(context.Background(), "admin", "pw")
	if err != nil || tok != "admin-tok" {
		t.Fatalf("tok=%s err=%v", tok, err)
	}
	if _, err := uc.AdminLogin(context.Background(), "admin", "bad"); err != ErrUnauthorized {
		t.Fatalf("got %v", err)
	}
}

func TestEthAuthorize_AutoPlacesByInvite(t *testing.T) {
	users := newMemUsers()
	place := NewPlacementUseCase(users, newMemPlacements(), nil)
	uc := NewUserUseCase(users, &memRecommends{path: map[uint64]string{}}, stubVerifier{}, stubTokens{}, &conf.Auth{}, addrGenesis, place)
	if _, err := uc.EthAuthorize(context.Background(), addrGenesis, "sig", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.EthAuthorize(context.Background(), addrUserB, "sig", addrGenesis); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.EthAuthorize(context.Background(), addrUserC, "sig", addrGenesis); err != nil {
		t.Fatal(err)
	}
	genesis, err := users.FindByAddress(context.Background(), addrGenesis)
	if err != nil {
		t.Fatal(err)
	}
	b, err := users.FindByAddress(context.Background(), addrUserB)
	if err != nil {
		t.Fatal(err)
	}
	c, err := users.FindByAddress(context.Background(), addrUserC)
	if err != nil {
		t.Fatal(err)
	}
	assertPlaced(t, place, b.ID, genesis.ID, SideLeft)
	assertPlaced(t, place, c.ID, genesis.ID, SideRight)
	if p, err := place.GetByUser(context.Background(), genesis.ID); err != nil || p != nil {
		t.Fatalf("genesis placed: %+v %v", p, err)
	}
}
