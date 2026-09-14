package biz

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/cigc/app/internal/pkg/money"
	"github.com/cigc/app/internal/pkg/wallet"
	"github.com/shopspring/decimal"
)

const (
	SideLeft  = "L"
	SideRight = "R"

	maxSharedChainWalk = 4096
	placeConflictRetry = 3
)

// Placement 是会员在安置树上的位置（与推荐关系分离）。
type Placement struct {
	ID        uint64
	UserID    uint64
	SponsorID uint64
	Side      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PlacementView 管理端展示行。
type PlacementView struct {
	Placement
	UserAddress    string
	SponsorAddress string
}

// PlacementRepo 安置读写。
type PlacementRepo interface {
	Create(ctx context.Context, p *Placement) (*Placement, error)
	FindByUserID(ctx context.Context, userID uint64) (*Placement, error)
	FindChild(ctx context.Context, sponsorID uint64, side string) (*Placement, error)
	ListBySponsor(ctx context.Context, sponsorID uint64) ([]*Placement, error)
	ListPaged(ctx context.Context, address string, page, pageSize int) ([]*PlacementView, int, error)
	ListAll(ctx context.Context) ([]*Placement, error)
}

// RecommendNode 安置直接子及其子树累计已支付。
type RecommendNode struct {
	UserID  uint64
	Address string
	Amount  decimal.Decimal
	Side    string
	Left    *RecommendNode
	Right   *RecommendNode
}

// DownlinePerson 管理端查看下级的一人摘要。
type DownlinePerson struct {
	UserID    uint64
	Address   string
	Paid      decimal.Decimal
	CreatedAt time.Time
	Children  []*DownlinePerson
}

// DownlineView 当前会员一层：邀请直推名单 + 左右安置；Full 时递归整树。
type DownlineView struct {
	Current     *DownlinePerson
	Inviter     *DownlinePerson
	Sponsor     *DownlinePerson
	SponsorSide string
	Invites     []*DownlinePerson
	Left        *RecommendNode
	Right       *RecommendNode
	Full        bool
}

// TeamVolume 左右区结余汇总（min=小区 max=大区）。
type TeamVolume struct {
	Left  decimal.Decimal
	Right decimal.Decimal
	Min   decimal.Decimal
	Max   decimal.Decimal
	Total decimal.Decimal
}

// ZoneVolume 由左右结余得到总/大区/小区。
func ZoneVolume(left, right decimal.Decimal) TeamVolume {
	left = money.Round(left)
	right = money.Round(right)
	v := TeamVolume{Left: left, Right: right, Total: money.Round(left.Add(right))}
	if left.LessThan(right) {
		v.Min, v.Max = left, right
	} else {
		v.Min, v.Max = right, left
	}
	return v
}

// PlacementUseCase 安置落位与树查询。
type PlacementUseCase struct {
	users      UserRepo
	placements PlacementRepo
	matches    MatchRepo
}

// NewPlacementUseCase 构造安置用例。matches 可为 nil（不读结余）。
func NewPlacementUseCase(users UserRepo, placements PlacementRepo, matches MatchRepo) *PlacementUseCase {
	return &PlacementUseCase{users: users, placements: placements, matches: matches}
}

// NormalizeSide 归一化为 L/R。
func NormalizeSide(side string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(side)) {
	case SideLeft, "LEFT", "左":
		return SideLeft, nil
	case SideRight, "RIGHT", "右":
		return SideRight, nil
	default:
		return "", ErrPlacementInvalidSide
	}
}

// Place 将 user 挂到 sponsor 的指定侧直接子位；该侧已有子或用户已安置则冲突。
func (uc *PlacementUseCase) Place(ctx context.Context, userID, sponsorID uint64, side string) (*Placement, error) {
	side, err := NormalizeSide(side)
	if err != nil {
		return nil, err
	}
	if userID == 0 || sponsorID == 0 {
		return nil, ErrInvalidAmount
	}
	if userID == sponsorID {
		return nil, ErrPlacementSelf
	}
	if _, err := uc.users.FindByID(ctx, userID); err != nil {
		return nil, err
	}
	if _, err := uc.users.FindByID(ctx, sponsorID); err != nil {
		return nil, err
	}
	if existing, err := uc.placements.FindByUserID(ctx, userID); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, ErrPlacementConflict
	}
	if child, err := uc.placements.FindChild(ctx, sponsorID, side); err != nil {
		return nil, err
	} else if child != nil {
		return nil, ErrPlacementConflict
	}
	return uc.placements.Create(ctx, &Placement{
		UserID:    userID,
		SponsorID: sponsorID,
		Side:      side,
	})
}

// PlaceByInvite 按邀请时间自动落位。创世不落位；已安置则原样返回。
// 规则见 CONTEXT.md「安置」。
func (uc *PlacementUseCase) PlaceByInvite(ctx context.Context, userID uint64) (*Placement, error) {
	if userID == 0 {
		return nil, ErrInvalidAmount
	}
	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.InviterID == nil {
		return nil, nil
	}
	if existing, err := uc.placements.FindByUserID(ctx, userID); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}

	var last error
	for i := 0; i < placeConflictRetry; i++ {
		sponsorID, side, err := uc.findInviteSlot(ctx, *user.InviterID)
		if err != nil {
			return nil, err
		}
		p, err := uc.Place(ctx, userID, sponsorID, side)
		if err == nil {
			return p, nil
		}
		if !errors.Is(err, ErrPlacementConflict) {
			return nil, err
		}
		last = err
	}
	return nil, last
}

func (uc *PlacementUseCase) findInviteSlot(ctx context.Context, inviterID uint64) (uint64, string, error) {
	left, err := uc.placements.FindChild(ctx, inviterID, SideLeft)
	if err != nil {
		return 0, "", err
	}
	if left == nil {
		return inviterID, SideLeft, nil
	}
	right, err := uc.placements.FindChild(ctx, inviterID, SideRight)
	if err != nil {
		return 0, "", err
	}
	if right == nil {
		return inviterID, SideRight, nil
	}

	leftCur, rightCur := left.UserID, right.UserID
	for i := 0; i < maxSharedChainWalk; i++ {
		progress := false
		taken, next, err := uc.trySharedLeft(ctx, leftCur)
		if err != nil {
			return 0, "", err
		}
		if taken {
			return leftCur, SideLeft, nil
		}
		if next != 0 {
			leftCur = next
			progress = true
		}

		taken, next, err = uc.trySharedLeft(ctx, rightCur)
		if err != nil {
			return 0, "", err
		}
		if taken {
			return rightCur, SideLeft, nil
		}
		if next != 0 {
			rightCur = next
			progress = true
		}
		if !progress {
			return 0, "", ErrPlacementNoSlot
		}
	}
	return 0, "", ErrPlacementNoSlot
}

// trySharedLeft 查看 node 左区（共享链）。空则可占用；已有人则沿其左区继续（不论谁邀请的）。
func (uc *PlacementUseCase) trySharedLeft(ctx context.Context, nodeID uint64) (take bool, next uint64, err error) {
	child, err := uc.placements.FindChild(ctx, nodeID, SideLeft)
	if err != nil {
		return false, 0, err
	}
	if child == nil {
		return true, 0, nil
	}
	return false, child.UserID, nil
}

// GetByUser 查某人安置；没有则 nil。
func (uc *PlacementUseCase) GetByUser(ctx context.Context, userID uint64) (*PlacementView, error) {
	p, err := uc.placements.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}
	return uc.toView(ctx, p)
}

// ListChildren 返回某 sponsor 的直接左右子（0～2 条）。
func (uc *PlacementUseCase) ListChildren(ctx context.Context, sponsorID uint64) ([]*PlacementView, error) {
	rows, err := uc.placements.ListBySponsor(ctx, sponsorID)
	if err != nil {
		return nil, err
	}
	out := make([]*PlacementView, 0, len(rows))
	for _, p := range rows {
		v, err := uc.toView(ctx, p)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// ListAdmin 安置分页，可按被安置人或上级地址模糊筛。
func (uc *PlacementUseCase) ListAdmin(ctx context.Context, address string, page int) ([]*PlacementView, int, error) {
	if page < 1 {
		page = 1
	}
	return uc.placements.ListPaged(ctx, address, page, DefaultRewardPageSize)
}

func (uc *PlacementUseCase) toView(ctx context.Context, p *Placement) (*PlacementView, error) {
	v := &PlacementView{Placement: *p}
	if u, err := uc.users.FindByID(ctx, p.UserID); err == nil {
		v.UserAddress = u.Address
	} else if err != ErrUserNotFound {
		return nil, err
	}
	if s, err := uc.users.FindByID(ctx, p.SponsorID); err == nil {
		v.SponsorAddress = s.Address
	} else if err != ErrUserNotFound {
		return nil, err
	}
	return v, nil
}

// ResolveUserID 按 id 或地址解析用户。
func (uc *PlacementUseCase) ResolveUserID(ctx context.Context, id uint64, address string) (uint64, error) {
	if id > 0 {
		if _, err := uc.users.FindByID(ctx, id); err != nil {
			return 0, err
		}
		return id, nil
	}
	address = strings.TrimSpace(address)
	if address == "" {
		return 0, ErrInvalidAmount
	}
	norm, ok := wallet.NormalizeAddress(address)
	if !ok {
		norm = strings.ToLower(address)
	}
	u, err := uc.users.FindByAddress(ctx, norm)
	if err != nil {
		return 0, err
	}
	return u.ID, nil
}

// TeamVolumeOf 读对碰结余并汇总；无结余行则全 0。
func (uc *PlacementUseCase) TeamVolumeOf(ctx context.Context, userID uint64) (TeamVolume, error) {
	if uc.matches == nil || userID == 0 {
		return ZoneVolume(decimal.Zero, decimal.Zero), nil
	}
	bal, err := uc.matches.Get(ctx, userID)
	if err != nil {
		return TeamVolume{}, err
	}
	if bal == nil {
		return ZoneVolume(decimal.Zero, decimal.Zero), nil
	}
	return ZoneVolume(bal.LeftRemain, bal.RightRemain), nil
}

// InSubtree 判断 target 是否为自己或安置后代。
func (uc *PlacementUseCase) InSubtree(ctx context.Context, rootID, targetID uint64) (bool, error) {
	if rootID == 0 || targetID == 0 {
		return false, nil
	}
	if rootID == targetID {
		return true, nil
	}
	cur := targetID
	seen := map[uint64]struct{}{}
	for i := 0; i < maxSharedChainWalk; i++ {
		if _, ok := seen[cur]; ok {
			return false, nil
		}
		seen[cur] = struct{}{}
		p, err := uc.placements.FindByUserID(ctx, cur)
		if err != nil {
			return false, err
		}
		if p == nil {
			return false, nil
		}
		if p.SponsorID == rootID {
			return true, nil
		}
		cur = p.SponsorID
	}
	return false, nil
}

// ListRecommend 用户端：address 空则查自己的直接子；查他人仅当对方在自己子树内。
func (uc *PlacementUseCase) ListRecommend(ctx context.Context, viewerID uint64, address string) ([]*RecommendNode, error) {
	targetID := viewerID
	address = strings.TrimSpace(address)
	if address != "" {
		id, err := uc.ResolveUserID(ctx, 0, address)
		if err != nil {
			return []*RecommendNode{}, nil
		}
		ok, err := uc.InSubtree(ctx, viewerID, id)
		if err != nil {
			return nil, err
		}
		if !ok {
			return []*RecommendNode{}, nil
		}
		targetID = id
	}
	if targetID == 0 {
		return []*RecommendNode{}, nil
	}
	return uc.listDirectRecommend(ctx, targetID)
}

const maxDownlineDepth = 64

// ExpandRecommendTrees 给直接子挂上整棵安置后代（测试全量展示）。
func (uc *PlacementUseCase) ExpandRecommendTrees(ctx context.Context, nodes []*RecommendNode) error {
	if len(nodes) == 0 {
		return nil
	}
	kids, paid, err := uc.placementKidsAndPaid(ctx)
	if err != nil {
		return err
	}
	seen := map[uint64]struct{}{}
	for _, n := range nodes {
		if err := uc.attachPlacementChildren(ctx, n, kids, paid, seen, 0); err != nil {
			return err
		}
	}
	return nil
}

func (uc *PlacementUseCase) placementKidsAndPaid(ctx context.Context) (map[uint64][]*Placement, map[uint64]decimal.Decimal, error) {
	rows, err := uc.placements.ListAll(ctx)
	if err != nil {
		return nil, nil, err
	}
	kids := map[uint64][]*Placement{}
	for _, p := range rows {
		if p == nil {
			continue
		}
		kids[p.SponsorID] = append(kids[p.SponsorID], p)
	}
	paid, err := uc.subtreePaidMap(ctx)
	if err != nil {
		return nil, nil, err
	}
	return kids, paid, nil
}

func (uc *PlacementUseCase) attachPlacementChildren(ctx context.Context, node *RecommendNode, kids map[uint64][]*Placement, paid map[uint64]decimal.Decimal, seen map[uint64]struct{}, depth int) error {
	if node == nil || depth >= maxDownlineDepth {
		return nil
	}
	if _, ok := seen[node.UserID]; ok {
		return nil
	}
	seen[node.UserID] = struct{}{}
	for _, p := range kids[node.UserID] {
		u, err := uc.users.FindByID(ctx, p.UserID)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				continue
			}
			return err
		}
		child := &RecommendNode{
			UserID:  p.UserID,
			Address: u.Address,
			Amount:  money.Round(paid[p.UserID]),
			Side:    p.Side,
		}
		switch p.Side {
		case SideLeft:
			node.Left = child
		case SideRight:
			node.Right = child
		}
		if err := uc.attachPlacementChildren(ctx, child, kids, paid, seen, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (uc *PlacementUseCase) inviteSubtree(ctx context.Context, userID uint64, seen map[uint64]struct{}, depth int) ([]*DownlinePerson, error) {
	if depth >= maxDownlineDepth {
		return nil, nil
	}
	invitees, err := uc.users.ListByInviter(ctx, userID)
	if err != nil {
		return nil, err
	}
	sort.Slice(invitees, func(i, j int) bool { return invitees[i].ID < invitees[j].ID })
	out := make([]*DownlinePerson, 0, len(invitees))
	for _, it := range invitees {
		if it == nil {
			continue
		}
		if _, ok := seen[it.ID]; ok {
			continue
		}
		seen[it.ID] = struct{}{}
		p := downlinePerson(it)
		kids, err := uc.inviteSubtree(ctx, it.ID, seen, depth+1)
		if err != nil {
			return nil, err
		}
		p.Children = kids
		out = append(out, p)
	}
	return out, nil
}

// ListRecommendAdmin 管理端可查任意地址的直接子；无地址则空列表。
func (uc *PlacementUseCase) ListRecommendAdmin(ctx context.Context, address string) ([]*RecommendNode, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return []*RecommendNode{}, nil
	}
	id, err := uc.ResolveUserID(ctx, 0, address)
	if err != nil {
		return []*RecommendNode{}, nil
	}
	return uc.listDirectRecommend(ctx, id)
}

func (uc *PlacementUseCase) listDirectRecommend(ctx context.Context, sponsorID uint64) ([]*RecommendNode, error) {
	children, err := uc.placements.ListBySponsor(ctx, sponsorID)
	if err != nil {
		return nil, err
	}
	sort.Slice(children, func(i, j int) bool {
		if children[i].Side == children[j].Side {
			return children[i].UserID < children[j].UserID
		}
		return children[i].Side < children[j].Side
	})
	paid, err := uc.subtreePaidMap(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*RecommendNode, 0, len(children))
	for _, c := range children {
		u, err := uc.users.FindByID(ctx, c.UserID)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				continue
			}
			return nil, err
		}
		out = append(out, &RecommendNode{
			UserID:  c.UserID,
			Address: u.Address,
			Amount:  money.Round(paid[c.UserID]),
			Side:    c.Side,
		})
	}
	return out, nil
}

func (uc *PlacementUseCase) subtreePaidMap(ctx context.Context) (map[uint64]decimal.Decimal, error) {
	rows, err := uc.placements.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	kids := map[uint64][]uint64{}
	ids := map[uint64]struct{}{}
	for _, p := range rows {
		if p == nil {
			continue
		}
		kids[p.SponsorID] = append(kids[p.SponsorID], p.UserID)
		ids[p.UserID] = struct{}{}
		ids[p.SponsorID] = struct{}{}
	}
	all, err := uc.users.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	paid := map[uint64]decimal.Decimal{}
	for _, u := range all {
		if u == nil {
			continue
		}
		paid[u.ID] = money.Round(u.PaidAmount)
	}
	memo := map[uint64]decimal.Decimal{}
	visiting := map[uint64]struct{}{}
	var walk func(uint64) decimal.Decimal
	walk = func(id uint64) decimal.Decimal {
		if v, ok := memo[id]; ok {
			return v
		}
		if _, ok := visiting[id]; ok {
			return paid[id]
		}
		visiting[id] = struct{}{}
		sum := paid[id]
		for _, kid := range kids[id] {
			sum = sum.Add(walk(kid))
		}
		delete(visiting, id)
		sum = money.Round(sum)
		memo[id] = sum
		return sum
	}
	for id := range ids {
		walk(id)
	}
	return memo, nil
}

func downlinePerson(u *User) *DownlinePerson {
	if u == nil {
		return nil
	}
	return &DownlinePerson{
		UserID:    u.ID,
		Address:   u.Address,
		Paid:      money.Round(u.PaidAmount),
		CreatedAt: u.CreatedAt,
	}
}

// InInviteSubtree 判断 target 是否为 ancestor 自己或其邀请后代。沿邀请链向上走，避免扫整棵邀请树。
func (uc *PlacementUseCase) InInviteSubtree(ctx context.Context, ancestorID, targetID uint64) (bool, error) {
	if ancestorID == 0 || targetID == 0 {
		return false, nil
	}
	if ancestorID == targetID {
		return true, nil
	}
	cur := targetID
	seen := map[uint64]struct{}{}
	for i := 0; i < maxSharedChainWalk; i++ {
		if _, ok := seen[cur]; ok {
			return false, nil
		}
		seen[cur] = struct{}{}
		u, err := uc.users.FindByID(ctx, cur)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				return false, nil
			}
			return false, err
		}
		if u == nil || u.InviterID == nil {
			return false, nil
		}
		if *u.InviterID == ancestorID {
			return true, nil
		}
		cur = *u.InviterID
	}
	return false, nil
}

// AdminDownline 管理端查看一人的邀请直推与左右安置，不递归整树。
func (uc *PlacementUseCase) AdminDownline(ctx context.Context, userID uint64, address string) (*DownlineView, error) {
	return uc.adminDownline(ctx, userID, address, false)
}

// AdminDownlineFull 测试用：展开全部邀请后代与双轨安置。
func (uc *PlacementUseCase) AdminDownlineFull(ctx context.Context, userID uint64, address string) (*DownlineView, error) {
	return uc.adminDownline(ctx, userID, address, true)
}

func (uc *PlacementUseCase) adminDownline(ctx context.Context, userID uint64, address string, full bool) (*DownlineView, error) {
	id, err := uc.ResolveUserID(ctx, userID, address)
	if err != nil {
		return nil, err
	}
	u, err := uc.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	view := &DownlineView{Current: downlinePerson(u), Invites: []*DownlinePerson{}, Full: full}
	if u.InviterID != nil {
		if inv, err := uc.users.FindByID(ctx, *u.InviterID); err == nil {
			view.Inviter = downlinePerson(inv)
		} else if !errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
	}
	if p, err := uc.placements.FindByUserID(ctx, id); err != nil {
		return nil, err
	} else if p != nil {
		view.SponsorSide = p.Side
		if s, err := uc.users.FindByID(ctx, p.SponsorID); err == nil {
			view.Sponsor = downlinePerson(s)
		} else if !errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
	}
	invitees, err := uc.users.ListByInviter(ctx, id)
	if err != nil {
		return nil, err
	}
	sort.Slice(invitees, func(i, j int) bool { return invitees[i].ID < invitees[j].ID })
	for _, it := range invitees {
		view.Invites = append(view.Invites, downlinePerson(it))
	}
	children, err := uc.listDirectRecommend(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, c := range children {
		cp := *c
		switch c.Side {
		case SideLeft:
			view.Left = &cp
		case SideRight:
			view.Right = &cp
		}
	}
	if !full {
		return view, nil
	}
	seenInv := map[uint64]struct{}{id: {}}
	for _, it := range view.Invites {
		if it == nil {
			continue
		}
		seenInv[it.UserID] = struct{}{}
		kids, err := uc.inviteSubtree(ctx, it.UserID, seenInv, 1)
		if err != nil {
			return nil, err
		}
		it.Children = kids
	}
	kids, paid, err := uc.placementKidsAndPaid(ctx)
	if err != nil {
		return nil, err
	}
	seenPlace := map[uint64]struct{}{id: {}}
	if err := uc.attachPlacementChildren(ctx, view.Left, kids, paid, seenPlace, 0); err != nil {
		return nil, err
	}
	if err := uc.attachPlacementChildren(ctx, view.Right, kids, paid, seenPlace, 0); err != nil {
		return nil, err
	}
	return view, nil
}

// UserDownline 用户端查看邀请直推与左右安置，只返回一层；address 仅允许自己或邀请/安置子树内的人。
func (uc *PlacementUseCase) UserDownline(ctx context.Context, viewerID uint64, address string) (*DownlineView, error) {
	if viewerID == 0 {
		return nil, ErrUnauthorized
	}
	targetID := viewerID
	address = strings.TrimSpace(address)
	if address != "" {
		id, err := uc.ResolveUserID(ctx, 0, address)
		if err != nil {
			return nil, err
		}
		inInvite, err := uc.InInviteSubtree(ctx, viewerID, id)
		if err != nil {
			return nil, err
		}
		inPlace, err := uc.InSubtree(ctx, viewerID, id)
		if err != nil {
			return nil, err
		}
		if !inInvite && !inPlace {
			return nil, ErrForbidden
		}
		targetID = id
	}
	return uc.AdminDownline(ctx, targetID, "")
}
