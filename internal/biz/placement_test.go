package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
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
	uc := NewPlacementUseCase(users, newMemPlacements(), nil)

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
	uc := NewPlacementUseCase(users, newMemPlacements(), nil)

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
	uc := NewPlacementUseCase(users, newMemPlacements(), nil)
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
	uc := NewPlacementUseCase(users, newMemPlacements(), nil)
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
	uc := NewPlacementUseCase(users, newMemPlacements(), nil)
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
	uc := NewPlacementUseCase(users, newMemPlacements(), nil)
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
	uc := NewPlacementUseCase(users, newMemPlacements(), nil)
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
	uc := NewPlacementUseCase(users, newMemPlacements(), nil)
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
	uc := NewPlacementUseCase(users, newMemPlacements(), nil)
	b := inviteAndPlace(t, users, uc, "0xb", a.ID)
	p1, err := uc.PlaceByInvite(context.Background(), b.ID)
	if err != nil || p1 == nil || p1.SponsorID != a.ID {
		t.Fatalf("%+v %v", p1, err)
	}
}

func TestZoneVolume(t *testing.T) {
	v := ZoneVolume(decimal.RequireFromString("400"), decimal.RequireFromString("1000"))
	if !v.Min.Equal(decimal.RequireFromString("400")) || !v.Max.Equal(decimal.RequireFromString("1000")) || !v.Total.Equal(decimal.RequireFromString("1400")) {
		t.Fatalf("%+v", v)
	}
	v = ZoneVolume(decimal.Zero, decimal.Zero)
	if !v.Min.IsZero() || !v.Max.IsZero() || !v.Total.IsZero() {
		t.Fatalf("%+v", v)
	}
}

func TestPairedVolume(t *testing.T) {
	got := PairedVolume(
		decimal.RequireFromString("500"),
		decimal.RequireFromString("50"),
		decimal.RequireFromString("100"),
		decimal.RequireFromString("50"),
	)
	if !got.Equal(decimal.RequireFromString("200")) {
		t.Fatalf("paired=%s", got)
	}
	if !PairedVolume(decimal.RequireFromString("10"), decimal.Zero, decimal.RequireFromString("20"), decimal.Zero).IsZero() {
		t.Fatal("negative pair should clamp to 0")
	}
}

func TestAdminTeamStats_SubtreeAndPair(t *testing.T) {
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{Address: "0xa", PaidAmount: decimal.RequireFromString("100")})
	if err != nil {
		t.Fatal(err)
	}
	b, err := users.Create(context.Background(), &User{Address: "0xb", PaidAmount: decimal.RequireFromString("200")})
	if err != nil {
		t.Fatal(err)
	}
	c, err := users.Create(context.Background(), &User{Address: "0xc", PaidAmount: decimal.RequireFromString("50")})
	if err != nil {
		t.Fatal(err)
	}
	d, err := users.Create(context.Background(), &User{Address: "0xd", PaidAmount: decimal.RequireFromString("300")})
	if err != nil {
		t.Fatal(err)
	}
	matches := newMemMatch()
	if err := matches.SaveRemains(context.Background(), a.ID, decimal.RequireFromString("100"), decimal.RequireFromString("50")); err != nil {
		t.Fatal(err)
	}
	uc := NewPlacementUseCase(users, newMemPlacements(), matches)
	if _, err := uc.Place(context.Background(), b.ID, a.ID, SideLeft); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Place(context.Background(), c.ID, a.ID, SideRight); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Place(context.Background(), d.ID, b.ID, SideLeft); err != nil {
		t.Fatal(err)
	}
	stats, err := uc.AdminTeamStats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got := stats[a.ID]
	if !got.Volume.Min.Equal(decimal.RequireFromString("50")) || !got.Volume.Max.Equal(decimal.RequireFromString("500")) || !got.Volume.Total.Equal(decimal.RequireFromString("550")) {
		t.Fatalf("A volume=%+v", got.Volume)
	}
	if !got.Paired.Equal(decimal.RequireFromString("200")) {
		t.Fatalf("A paired=%s", got.Paired)
	}
	if stats[a.ID].Volume.Total.Equal(decimal.RequireFromString("650")) {
		t.Fatal("total must exclude self paid_amount")
	}
}

func TestListRecommend_SubtreePaidAndScope(t *testing.T) {
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{Address: "0xa", PaidAmount: decimal.RequireFromString("100")})
	if err != nil {
		t.Fatal(err)
	}
	b, err := users.Create(context.Background(), &User{Address: "0xb", PaidAmount: decimal.RequireFromString("200")})
	if err != nil {
		t.Fatal(err)
	}
	c, err := users.Create(context.Background(), &User{Address: "0xc", PaidAmount: decimal.RequireFromString("50")})
	if err != nil {
		t.Fatal(err)
	}
	d, err := users.Create(context.Background(), &User{Address: "0xd", PaidAmount: decimal.RequireFromString("300")})
	if err != nil {
		t.Fatal(err)
	}
	outsider, err := users.Create(context.Background(), &User{Address: "0xee"})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewPlacementUseCase(users, newMemPlacements(), nil)
	if _, err := uc.Place(context.Background(), b.ID, a.ID, SideLeft); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Place(context.Background(), c.ID, a.ID, SideRight); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Place(context.Background(), d.ID, b.ID, SideLeft); err != nil {
		t.Fatal(err)
	}

	nodes, err := uc.ListRecommend(context.Background(), a.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 || nodes[0].Address != "0xb" || !nodes[0].Amount.Equal(decimal.RequireFromString("500")) {
		t.Fatalf("A children=%+v", nodes)
	}
	if nodes[1].Address != "0xc" || !nodes[1].Amount.Equal(decimal.RequireFromString("50")) {
		t.Fatalf("C amount=%s", nodes[1].Amount)
	}

	underB, err := uc.ListRecommend(context.Background(), a.ID, "0xb")
	if err != nil || len(underB) != 1 || underB[0].Address != "0xd" || !underB[0].Amount.Equal(decimal.RequireFromString("300")) {
		t.Fatalf("under B: %+v %v", underB, err)
	}
	empty, err := uc.ListRecommend(context.Background(), a.ID, outsider.Address)
	if err != nil || len(empty) != 0 {
		t.Fatalf("outsider: %+v %v", empty, err)
	}
	admin, err := uc.ListRecommendAdmin(context.Background(), "0xb")
	if err != nil || len(admin) != 1 || admin[0].Address != "0xd" {
		t.Fatalf("admin: %+v %v", admin, err)
	}
}

func TestAdminDownline_OneLevelInviteAndPlacement(t *testing.T) {
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{Address: "0xa", PaidAmount: decimal.RequireFromString("100")})
	if err != nil {
		t.Fatal(err)
	}
	b, err := users.Create(context.Background(), &User{Address: "0xb", InviterID: &a.ID, PaidAmount: decimal.RequireFromString("200")})
	if err != nil {
		t.Fatal(err)
	}
	c, err := users.Create(context.Background(), &User{Address: "0xc", InviterID: &a.ID, PaidAmount: decimal.RequireFromString("50")})
	if err != nil {
		t.Fatal(err)
	}
	d, err := users.Create(context.Background(), &User{Address: "0xd", InviterID: &b.ID, PaidAmount: decimal.RequireFromString("300")})
	if err != nil {
		t.Fatal(err)
	}
	uc := NewPlacementUseCase(users, newMemPlacements(), nil)
	if _, err := uc.Place(context.Background(), b.ID, a.ID, SideLeft); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Place(context.Background(), c.ID, a.ID, SideRight); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Place(context.Background(), d.ID, b.ID, SideLeft); err != nil {
		t.Fatal(err)
	}

	view, err := uc.AdminDownline(context.Background(), a.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if view.Current == nil || view.Current.Address != "0xa" {
		t.Fatalf("current %+v", view.Current)
	}
	if len(view.Invites) != 2 {
		t.Fatalf("invites=%d want 2 (not whole tree)", len(view.Invites))
	}
	if view.Left == nil || view.Left.Address != "0xb" || view.Right == nil || view.Right.Address != "0xc" {
		t.Fatalf("placement L=%+v R=%+v", view.Left, view.Right)
	}
	for _, it := range view.Invites {
		if it.Address == "0xd" {
			t.Fatal("must not include grandchild")
		}
	}

	child, err := uc.AdminDownline(context.Background(), 0, "0xb")
	if err != nil {
		t.Fatal(err)
	}
	if child.Inviter == nil || child.Inviter.Address != "0xa" {
		t.Fatalf("inviter %+v", child.Inviter)
	}
	if child.Sponsor == nil || child.Sponsor.Address != "0xa" || child.SponsorSide != SideLeft {
		t.Fatalf("sponsor %+v %s", child.Sponsor, child.SponsorSide)
	}
	if len(child.Invites) != 1 || child.Invites[0].Address != "0xd" {
		t.Fatalf("child invites %+v", child.Invites)
	}

	full, err := uc.AdminDownlineFull(context.Background(), a.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if !full.Full {
		t.Fatal("full flag")
	}
	var sawD bool
	for _, it := range full.Invites {
		if it.Address == "0xb" {
			for _, g := range it.Children {
				if g.Address == "0xd" {
					sawD = true
				}
			}
		}
	}
	if !sawD {
		t.Fatalf("full invite tree missing grandchild: %+v", full.Invites)
	}
	if full.Left == nil || full.Left.Left == nil || full.Left.Left.Address != "0xd" {
		t.Fatalf("full placement L.L=%+v", full.Left)
	}

	own, err := uc.UserDownline(context.Background(), a.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if own.Current == nil || own.Current.Address != "0xa" {
		t.Fatalf("own %+v", own.Current)
	}
	if own.Full {
		t.Fatal("user downline must be one level")
	}
	if len(own.Invites) != 2 {
		t.Fatalf("invites=%d want 2", len(own.Invites))
	}
	for _, it := range own.Invites {
		if it != nil && len(it.Children) != 0 {
			t.Fatal("must not expand invite children")
		}
	}
	if own.Left != nil && own.Left.Left != nil {
		t.Fatal("must not expand placement children")
	}
	if _, err := uc.UserDownline(context.Background(), a.ID, "0xd"); err != nil {
		t.Fatal(err)
	}
	outsider, err := users.Create(context.Background(), &User{Address: "0xz"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.UserDownline(context.Background(), a.ID, outsider.Address); !errors.Is(err, ErrForbidden) {
		t.Fatalf("outsider err=%v", err)
	}
}

func TestTeamVolumeOf(t *testing.T) {
	users := newMemUsers()
	a, err := users.Create(context.Background(), &User{Address: "0xa"})
	if err != nil {
		t.Fatal(err)
	}
	m := newMemMatch()
	if err := m.SaveRemains(context.Background(), a.ID, decimal.RequireFromString("400"), decimal.RequireFromString("1000")); err != nil {
		t.Fatal(err)
	}
	uc := NewPlacementUseCase(users, newMemPlacements(), m)
	v, err := uc.TeamVolumeOf(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Min.Equal(decimal.RequireFromString("400")) || !v.Max.Equal(decimal.RequireFromString("1000")) || !v.Total.Equal(decimal.RequireFromString("1400")) {
		t.Fatalf("%+v", v)
	}
}
