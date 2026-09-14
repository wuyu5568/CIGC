package biz

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/cigc/app/internal/conf"
	"github.com/cigc/app/internal/pkg/wallet"
	"github.com/shopspring/decimal"
)

// User 是会员主档。可用余额为内部奖励账户，不是链上钱包。
type User struct {
	ID               uint64
	Address          string
	InviterID        *uint64
	AvailableBalance decimal.Decimal
	RechargeBalance  decimal.Decimal
	FrozenBalance    decimal.Decimal
	FrozenIspay      decimal.Decimal
	IspayBalance     decimal.Decimal
	LockBalance      decimal.Decimal
	LockIspay        decimal.Decimal
	PaidAmount       decimal.Decimal
	CapEffective     decimal.Decimal
	DisabledAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// IsActivated 至少有一笔已支付订单。
func (u *User) IsActivated() bool {
	return u != nil && u.PaidAmount.IsPositive()
}

// IsDisabled 软删/锁定后不可登录。
func (u *User) IsDisabled() bool {
	return u != nil && u.DisabledAt != nil
}

// LoginChallenge 一次性登录 nonce（v1 接口预留，主路径 eth_authorize 不用）。
type LoginChallenge struct {
	ID        uint64
	Address   string
	Nonce     string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// UserRepo 用户读写。
type UserRepo interface {
	FindByAddress(ctx context.Context, address string) (*User, error)
	FindByID(ctx context.Context, id uint64) (*User, error)
	Create(ctx context.Context, u *User) (*User, error)
	AddPaidAmount(ctx context.Context, userID uint64, delta decimal.Decimal) error
	ListAll(ctx context.Context) ([]*User, error)
	SetCapEffective(ctx context.Context, userID uint64, cap decimal.Decimal) error
	ListAdmin(ctx context.Context, address string, page, pageSize int) ([]*User, int, error)
	ListByInviter(ctx context.Context, inviterID uint64) ([]*User, error)
	SetDisabledAt(ctx context.Context, userID uint64, disabledAt *time.Time) error
}

// AdminUserRow 管理端会员行。
type AdminUserRow struct {
	User
	InviterAddress string
}

// AdminUserPage 管理端会员分页。
type AdminUserPage struct {
	Items []*AdminUserRow
	Total int
}

// UserBalanceRepo 变更内部账户余额。
type UserBalanceRepo interface {
	AddAvailableBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error
	SubAvailableBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error
	AddRechargeBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error
	SubRechargeBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error
	AddFrozenBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error
	SubFrozenBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error
	AddFrozenIspay(ctx context.Context, userID uint64, delta decimal.Decimal) error
	SubFrozenIspay(ctx context.Context, userID uint64, delta decimal.Decimal) error
	AddIspayBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error
	SubIspayBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error
	AddLockBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error
	SubLockBalance(ctx context.Context, userID uint64, delta decimal.Decimal) error
	AddLockIspay(ctx context.Context, userID uint64, delta decimal.Decimal) error
	SubLockIspay(ctx context.Context, userID uint64, delta decimal.Decimal) error
}

// RecommendRepo 物化推荐祖先 path，供管理奖向上查找使用。
type RecommendRepo interface {
	GetPath(ctx context.Context, userID uint64) (string, error)
	SavePath(ctx context.Context, userID uint64, path string) error
}

// LoginChallengeRepo nonce 挑战存储。
type LoginChallengeRepo interface {
	Create(ctx context.Context, address, nonce string, expiresAt time.Time) error
	FindUsable(ctx context.Context, address, nonce string, now time.Time) (*LoginChallenge, error)
	MarkUsed(ctx context.Context, id uint64, usedAt time.Time) error
}

// UserUseCase 钱包登录、邀请绑定、资料查询、管理登录。
type UserUseCase struct {
	users       UserRepo
	recommends  RecommendRepo
	verifier    SignatureVerifier
	tokens      TokenIssuer
	auth        *conf.Auth
	genesisAddr string
	place       *PlacementUseCase
}

// NewUserUseCase 构造用户用例。
func NewUserUseCase(
	users UserRepo,
	recommends RecommendRepo,
	verifier SignatureVerifier,
	tokens TokenIssuer,
	auth *conf.Auth,
	genesisAddr string,
	place *PlacementUseCase,
) *UserUseCase {
	return &UserUseCase{
		users:       users,
		recommends:  recommends,
		verifier:    verifier,
		tokens:      tokens,
		auth:        auth,
		genesisAddr: wallet.NormalizeOrEmpty(genesisAddr),
		place:       place,
	}
}

// EthAuthorize 校验对地址本身的 personal_sign；首次登录须绑定邀请人（创世除外）。
func (uc *UserUseCase) EthAuthorize(ctx context.Context, address, signature, inviteCode string) (*LoginResult, error) {
	rawAddr := strings.TrimSpace(address)
	address, ok := wallet.NormalizeAddress(address)
	if !ok || signature == "" {
		return nil, ErrInvalidSignature
	}
	rawInvite := strings.TrimSpace(inviteCode)
	inviteCode = wallet.NormalizeInviteCode(inviteCode)

	if err := uc.verifier.Verify(address, rawAddr, signature); err != nil {
		return nil, ErrInvalidSignature
	}

	user, err := uc.users.FindByAddress(ctx, address)
	if err == nil && user != nil {
		if user.IsDisabled() {
			return nil, ErrUserDisabled
		}
		token, err := uc.tokens.Issue(user.ID, user.Address)
		if err != nil {
			return nil, err
		}
		return &LoginResult{Token: token, User: user}, nil
	}
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	var inviterID *uint64
	isGenesis := uc.genesisAddr != "" && address == uc.genesisAddr
	if !isGenesis {
		if rawInvite == "" {
			return nil, ErrInviteRequired
		}
		if inviteCode == "" || inviteCode == address {
			return nil, ErrInviteInvalid
		}
		inviter, err := uc.users.FindByAddress(ctx, inviteCode)
		if err != nil || inviter == nil {
			return nil, ErrInviteInvalid
		}
		id := inviter.ID
		inviterID = &id
	}

	user, err = uc.users.Create(ctx, &User{Address: address, InviterID: inviterID})
	if err != nil {
		return nil, err
	}

	path := strconv.FormatUint(user.ID, 10)
	if inviterID != nil {
		parentPath, pathErr := uc.recommends.GetPath(ctx, *inviterID)
		if pathErr != nil {
			return nil, pathErr
		}
		if parentPath != "" {
			path = parentPath + "," + path
		}
	}
	if err := uc.recommends.SavePath(ctx, user.ID, path); err != nil {
		return nil, err
	}
	if err := uc.autoPlace(ctx, user.ID); err != nil {
		return nil, err
	}

	token, err := uc.tokens.Issue(user.ID, user.Address)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Token: token, User: user}, nil
}

func (uc *UserUseCase) autoPlace(ctx context.Context, userID uint64) error {
	if uc.place == nil {
		return nil
	}
	_, err := uc.place.PlaceByInvite(ctx, userID)
	return err
}

// GetProfile 按 ID 取用户。
func (uc *UserUseCase) GetProfile(ctx context.Context, userID uint64) (*User, error) {
	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.IsDisabled() {
		return nil, ErrUserDisabled
	}
	return user, nil
}

// AdminLogin 校验配置中的管理员账号并发放管理 JWT。
func (uc *UserUseCase) AdminLogin(_ context.Context, username, password string) (string, error) {
	if uc.auth == nil || username == "" || password == "" {
		return "", ErrUnauthorized
	}
	if username != uc.auth.AdminUsername || password != uc.auth.AdminPassword {
		return "", ErrUnauthorized
	}
	return uc.tokens.IssueAdmin()
}

// InviterAddress 查询邀请人钱包地址，没有则返回空串。
func (uc *UserUseCase) InviterAddress(ctx context.Context, user *User) (string, error) {
	if user == nil || user.InviterID == nil {
		return "", nil
	}
	inv, err := uc.users.FindByID(ctx, *user.InviterID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", nil
		}
		return "", err
	}
	return inv.Address, nil
}

// ListAdminUsers 管理端会员分页，可按地址模糊筛。
func (uc *UserUseCase) ListAdminUsers(ctx context.Context, address string, page int) (*AdminUserPage, error) {
	if page < 1 {
		page = 1
	}
	users, total, err := uc.users.ListAdmin(ctx, address, page, DefaultRewardPageSize)
	if err != nil {
		return nil, err
	}
	out := make([]*AdminUserRow, 0, len(users))
	for _, u := range users {
		invAddr, err := uc.InviterAddress(ctx, u)
		if err != nil {
			return nil, err
		}
		row := &AdminUserRow{User: *u, InviterAddress: invAddr}
		out = append(out, row)
	}
	return &AdminUserPage{Items: out, Total: total}, nil
}

// InviteCountMap 直推人数（按 inviter_id，不含安置子）。
func (uc *UserUseCase) InviteCountMap(ctx context.Context) (map[uint64]int, error) {
	out := map[uint64]int{}
	if uc == nil || uc.users == nil {
		return out, nil
	}
	all, err := uc.users.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	for _, u := range all {
		if u == nil || u.InviterID == nil || *u.InviterID == 0 {
			continue
		}
		out[*u.InviterID]++
	}
	return out, nil
}

// SetUserLock 锁定/解锁会员；locked=true 写入 disabled_at，false 清空。幂等。
func (uc *UserUseCase) SetUserLock(ctx context.Context, userID uint64, locked bool) error {
	if userID == 0 {
		return ErrInvalidAmount
	}
	u, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if locked {
		if u.IsDisabled() {
			return nil
		}
		now := time.Now()
		return uc.users.SetDisabledAt(ctx, userID, &now)
	}
	if !u.IsDisabled() {
		return nil
	}
	return uc.users.SetDisabledAt(ctx, userID, nil)
}

// UnlockUser 解锁会员（SetUserLock(false) 别名）。
func (uc *UserUseCase) UnlockUser(ctx context.Context, userID uint64) error {
	return uc.SetUserLock(ctx, userID, false)
}
