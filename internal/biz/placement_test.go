package biz

import (
	"context"
	"errors"
	"testing"
)

type memPlacements struct {
	byUser    map[uint64]*Placement
	bySponsor map[uint64]map[string]*Placement
	next      uint64
}

func newMemPlacements() *memPlacements {
	return &memPlacements{
		byUser:    map[uint64]*Placement{},
		bySponsor: map[uint64]map[string]*Placement{},
		next:      1,
	}
}

func (m *memPlacements) Create(_ context.Context, p *Placement) (*Placement, error) {
	if _, ok := m.byUser[p.UserID]; ok {
		return nil, ErrPlacementConflict
	}
	if m.bySponsor[p.SponsorID] == nil {
		m.bySponsor[p.SponsorID] = map[string]*Placement{}
	}
	if _, ok := m.bySponsor[p.SponsorID][p.Side]; ok {
		return nil, ErrPlacementConflict
	}
	cp := *p
	cp.ID = m.next
	m.next++
	m.byUser[cp.UserID] = &cp
	m.bySponsor[cp.SponsorID][cp.Side] = &cp
	out := cp
	return &out, nil
}

func (m *memPlacements) FindByUserID(_ context.Context, userID uint64) (*Placement, error) {
	p, ok := m.byUser[userID]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}

func (m *memPlacements) FindChild(_ context.Context, sponsorID uint64, side string) (*Placement, error) {
	sideMap := m.bySponsor[sponsorID]
	if sideMap == nil {
		return nil, nil
	}
	p, ok := sideMap[side]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}

func (m *memPlacements) ListBySponsor(_ context.Context, sponsorID uint64) ([]*Placement, error) {
	sideMap := m.bySponsor[sponsorID]
	var out []*Placement
	for _, p := range sideMap {
		cp := *p
		out = append(out, &cp)
	}
	return out, nil
}

func (m *memPlacements) ListPaged(_ context.Context, address string, page, pageSize int) ([]*PlacementView, int, error) {
	_ = address
	var all []*PlacementView
	for _, p := range m.byUser {
		cp := *p
		all = append(all, &PlacementView{Placement: cp})
	}
	total := len(all)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultRewardPageSize
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []*PlacementView{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return all[start:end], total, nil
}

func (m *memPlacements) ListAll(_ context.Context) ([]*Placement, error) {
	out := make([]*Placement, 0, len(m.byUser))
	for _, p := range m.byUser {
		cp := *p
		out = append(out, &cp)
	}
	return out, nil
}

func TestNormalizeSide(t *testing.T) {
	s, err := NormalizeSide("l")
	if err != nil || s != SideLeft {
		t.Fatalf("%s %v", s, err)
	}
	if _, err := NormalizeSide("x"); !errors.Is(err, ErrPlacementInvalidSide) {
		t.Fatalf("%v", err)
	}
}

func TestPlace_DirectSlots(t *testing.T) {
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{Address: "0xa"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := users.Create(context.Background(), &User{Address: "0xb"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := users.Create(context.Background(), &User{Address: "0xc"})
	if err != nil {
		t.Fatal(err)
	}
	d, err := users.Create(context.Background(), &User{Address: "0xd"})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewPlacementUseCase(users, newMemPlacements())

	if _, err := uc.Place(context.Background(), b.ID, a.ID, "L"); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Place(context.Background(), c.ID, a.ID, "R"); err != nil {
		t.Fatal(err)
	}
	// A 的左区已满
	if _, err := uc.Place(context.Background(), d.ID, a.ID, "L"); !errors.Is(err, ErrPlacementConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
	// 挂到 B 的左区（更深一层）
	if _, err := uc.Place(context.Background(), d.ID, b.ID, "L"); err != nil {
		t.Fatal(err)
	}
	// B 已安置，不能再挂
	if _, err := uc.Place(context.Background(), b.ID, c.ID, "L"); !errors.Is(err, ErrPlacementConflict) {
		t.Fatalf("re-place: %v", err)
	}
	if _, err := uc.Place(context.Background(), a.ID, a.ID, "L"); !errors.Is(err, ErrPlacementSelf) {
		t.Fatalf("self: %v", err)
	}

	children, err := uc.ListChildren(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 2 {
		t.Fatalf("children=%d", len(children))
	}
	view, err := uc.GetByUser(context.Background(), d.ID)
	if err != nil || view == nil || view.SponsorID != b.ID || view.Side != SideLeft {
		t.Fatalf("%+v %v", view, err)
	}
}

func inviteAndPlace(t *testing.T, users *memUsers, uc *PlacementUseCase, addr string, inviterID uint64) *User {
	t.Helper()
	id := inviterID
	u, err := users.Create(context.Background(), &User{Address: addr, InviterID: &id})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.PlaceByInvite(context.Background(), u.ID); err != nil {
		t.Fatal(err)
	}
	return u
}

func assertPlaced(t *testing.T, uc *PlacementUseCase, userID, sponsorID uint64, side string) {
	t.Helper()
	view, err := uc.GetByUser(context.Background(), userID)
	if err != nil || view == nil {
		t.Fatalf("user %d: %+v %v", userID, view, err)
	}
	if view.SponsorID != sponsorID || view.Side != side {
		t.Fatalf("user %d: sponsor=%d side=%s want %d %s", userID, view.SponsorID, view.Side, sponsorID, side)
	}
}

func TestPlaceByInvite_SharedChainOrder(t *testing.T) {
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{Address: "0xa"})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewPlacementUseCase(users, newMemPlacements())

	if p, err := uc.PlaceByInvite(context.Background(), a.ID); err != nil || p != nil {
		t.Fatalf("genesis: %+v %v", p, err)
	}

	b := inviteAndPlace(t, users, uc, "0xb", a.ID)
	c := inviteAndPlace(t, users, uc, "0xc", a.ID)
	d := inviteAndPlace(t, users, uc, "0xd", a.ID)
	e := inviteAndPlace(t, users, uc, "0xe", a.ID)
	f := inviteAndPlace(t, users, uc, "0xf", a.ID)
	g := inviteAndPlace(t, users, uc, "0xg", a.ID)

	assertPlaced(t, uc, b.ID, a.ID, SideLeft)
	assertPlaced(t, uc, c.ID, a.ID, SideRight)
	assertPlaced(t, uc, d.ID, b.ID, SideLeft)
	assertPlaced(t, uc, e.ID, c.ID, SideLeft)
	assertPlaced(t, uc, f.ID, d.ID, SideLeft)
	assertPlaced(t, uc, g.ID, e.ID, SideLeft)
}

func TestPlaceByInvite_InviteeUsesSharedFirst(t *testing.T) {
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{Address: "0xa"})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewPlacementUseCase(users, newMemPlacements())
	b := inviteAndPlace(t, users, uc, "0xb", a.ID)
	c := inviteAndPlace(t, users, uc, "0xc", a.ID)
	h := inviteAndPlace(t, users, uc, "0xh", b.ID)
	d := inviteAndPlace(t, users, uc, "0xd", a.ID)
	e := inviteAndPlace(t, users, uc, "0xe", a.ID)

	assertPlaced(t, uc, h.ID, b.ID, SideLeft)
	assertPlaced(t, uc, d.ID, c.ID, SideLeft)
	assertPlaced(t, uc, e.ID, h.ID, SideLeft)
}

func TestPlaceByInvite_InviteeOwnTree(t *testing.T) {
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{Address: "0xa"})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewPlacementUseCase(users, newMemPlacements())
	b := inviteAndPlace(t, users, uc, "0xb", a.ID)
	_ = inviteAndPlace(t, users, uc, "0xc", a.ID)
	h := inviteAndPlace(t, users, uc, "0xh", b.ID)
	i := inviteAndPlace(t, users, uc, "0xi", b.ID)
	j := inviteAndPlace(t, users, uc, "0xj", b.ID)
	k := inviteAndPlace(t, users, uc, "0xk", b.ID)

	assertPlaced(t, uc, h.ID, b.ID, SideLeft)
	assertPlaced(t, uc, i.ID, b.ID, SideRight)
	assertPlaced(t, uc, j.ID, h.ID, SideLeft)
	assertPlaced(t, uc, k.ID, i.ID, SideLeft)
}

func TestPlaceByInvite_RightTakenFirst_NextGoesUnderLeftChild(t *testing.T) {
	// 情况 5：C 先占 C 左，A 先填 B 左（E），再按左→右挂 E 左，不先钻 C 链。
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{Address: "0xa"})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewPlacementUseCase(users, newMemPlacements())
	b := inviteAndPlace(t, users, uc, "0xb", a.ID)
	c := inviteAndPlace(t, users, uc, "0xc", a.ID)
	h := inviteAndPlace(t, users, uc, "0xh", c.ID)
	e := inviteAndPlace(t, users, uc, "0xe", a.ID)
	f := inviteAndPlace(t, users, uc, "0xf", a.ID)

	assertPlaced(t, uc, h.ID, c.ID, SideLeft)
	assertPlaced(t, uc, e.ID, b.ID, SideLeft)
	assertPlaced(t, uc, f.ID, e.ID, SideLeft)
}

func TestPlaceByInvite_BothDepthFilled_GoesDeeperLeft(t *testing.T) {
	// E 左也被占时，A 的下一人挂 R 左（D 左上的人）。
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{Address: "0xa"})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewPlacementUseCase(users, newMemPlacements())
	b := inviteAndPlace(t, users, uc, "0xb", a.ID)
	_ = inviteAndPlace(t, users, uc, "0xc", a.ID)
	d := inviteAndPlace(t, users, uc, "0xd", b.ID)
	e := inviteAndPlace(t, users, uc, "0xe", a.ID)
	r := inviteAndPlace(t, users, uc, "0xr", d.ID)
	_ = inviteAndPlace(t, users, uc, "0xs", e.ID)
	f := inviteAndPlace(t, users, uc, "0xf", a.ID)

	assertPlaced(t, uc, r.ID, d.ID, SideLeft)
	assertPlaced(t, uc, f.ID, r.ID, SideLeft)
}

func TestPlaceByInvite_SkipOccupiedLeftThenRightThenDeeper(t *testing.T) {
	// 情况 10：D 在 B 左且 D 左已被 R 占 → A 先挂仍空的 E 左；E 左也有人才挂 R 左。
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{Address: "0xa"})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewPlacementUseCase(users, newMemPlacements())
	b := inviteAndPlace(t, users, uc, "0xb", a.ID)
	_ = inviteAndPlace(t, users, uc, "0xc", a.ID)
	d := inviteAndPlace(t, users, uc, "0xd", b.ID)
	e := inviteAndPlace(t, users, uc, "0xe", a.ID)
	r := inviteAndPlace(t, users, uc, "0xr", d.ID)
	f := inviteAndPlace(t, users, uc, "0xf", a.ID)

	assertPlaced(t, uc, r.ID, d.ID, SideLeft)
	assertPlaced(t, uc, f.ID, e.ID, SideLeft)

	g := inviteAndPlace(t, users, uc, "0xg", a.ID)
	assertPlaced(t, uc, g.ID, r.ID, SideLeft)
}

func TestPlaceByInvite_ContinuesThroughInviteeChild(t *testing.T) {
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{Address: "0xa"})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewPlacementUseCase(users, newMemPlacements())
	b := inviteAndPlace(t, users, uc, "0xb", a.ID)
	c := inviteAndPlace(t, users, uc, "0xc", a.ID)
	d := inviteAndPlace(t, users, uc, "0xd", b.ID)
	e := inviteAndPlace(t, users, uc, "0xe", a.ID)
	f := inviteAndPlace(t, users, uc, "0xf", a.ID)

	assertPlaced(t, uc, d.ID, b.ID, SideLeft)
	assertPlaced(t, uc, e.ID, c.ID, SideLeft)
	assertPlaced(t, uc, f.ID, d.ID, SideLeft)
}

func TestPlaceByInvite_Idempotent(t *testing.T) {
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{Address: "0xa"})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewPlacementUseCase(users, newMemPlacements())
	b := inviteAndPlace(t, users, uc, "0xb", a.ID)
	p1, err := uc.PlaceByInvite(context.Background(), b.ID)
	if err != nil || p1 == nil || p1.SponsorID != a.ID {
		t.Fatalf("%+v %v", p1, err)
	}
}
