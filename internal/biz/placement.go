package biz

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cigc/app/internal/pkg/wallet"
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

// PlacementUseCase 管理端手工落位（不做自动算法）。
type PlacementUseCase struct {
	users      UserRepo
	placements PlacementRepo
}

// NewPlacementUseCase 构造安置用例。
func NewPlacementUseCase(users UserRepo, placements PlacementRepo) *PlacementUseCase {
	return &PlacementUseCase{users: users, placements: placements}
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
