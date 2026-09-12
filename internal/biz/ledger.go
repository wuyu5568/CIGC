package biz

import (
	"context"
	"time"

	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
)

const (
	LedgerDirect      = "direct"
	LedgerMatch       = "match"
	LedgerManage      = "manage"
	LedgerAdminAdjust = "admin_adjust"
	LedgerWithdraw    = "withdraw"
	LedgerFreeze      = "withdraw_freeze"
	LedgerUnfreeze    = "withdraw_unfreeze"

	BalanceAvailable = "available"
	BalanceFrozen    = "frozen"

	// DefaultRewardPageSize 对齐 dapp 分页（每页 10）。
	DefaultRewardPageSize = 10
)

// NormalizeAmount 将业务金额对齐到金牛口径：Round(8)。
func NormalizeAmount(d decimal.Decimal) decimal.Decimal {
	return money.Round(d)
}

// LedgerEntry 是一条余额变动流水，禁止只改余额不写流水。
type LedgerEntry struct {
	ID          uint64
	UserID      uint64
	OrderID     *uint64
	EntryType   string
	Amount      decimal.Decimal
	BalanceKind string
	SettleDate  *time.Time
	Remark      string
	CreatedAt   time.Time
}

// LedgerRepo 持久化账本。
type LedgerRepo interface {
	Create(ctx context.Context, e *LedgerEntry) error
	ListByUser(ctx context.Context, userID uint64, from, to time.Time) ([]*LedgerEntry, error)
	ListPaged(ctx context.Context, address, entryType string, page, pageSize int) ([]*LedgerEntry, int, error)
	ExistsByOrderAndType(ctx context.Context, orderID uint64, entryType string) (bool, error)
	ExistsByUserTypeAndDate(ctx context.Context, userID uint64, entryType string, settleDate time.Time) (bool, error)
}

// RewardItem 是用户/管理端流水展示行（address/num 暂空，等管理奖 remark）。
type RewardItem struct {
	ID        uint64
	Amount    string
	Name      string
	Reason    string
	Address   string
	Num       string
	Remark    string
	CreatedAt time.Time
}

// RewardPage 分页结果。
type RewardPage struct {
	Items []*RewardItem
	Total int
}

// LedgerUseCase 账本查询（写入由奖励/提现模块负责）。
type LedgerUseCase struct {
	ledger LedgerRepo
	now    func() time.Time
}

// NewLedgerUseCase 构造账本查询用例。
func NewLedgerUseCase(ledger LedgerRepo) *LedgerUseCase {
	return &LedgerUseCase{ledger: ledger, now: time.Now}
}

// IsRewardEntry 三类收益流水；默认列表只含这些类型。
func IsRewardEntry(entryType string) bool {
	switch entryType {
	case LedgerDirect, LedgerMatch, LedgerManage:
		return true
	default:
		return false
	}
}

// ReqTypeToEntryType 将 dapp reqType 映射为 entry_type。
// 空或 "1"：默认三类收益（返回 ok=true 且 entryType="" 表示全部收益类）。
// 未知码：ok=false，调用方应返回空列表。
func ReqTypeToEntryType(reqType string) (entryType string, ok bool) {
	switch reqType {
	case "", "1":
		return "", true
	case "3":
		return LedgerDirect, true
	case "4":
		return LedgerMatch, true
	case "5":
		return LedgerManage, true
	default:
		return "", false
	}
}

// ReasonToEntryType 管理端 reason 筛选 → entry_type；空表示不按类型筛。
func ReasonToEntryType(reason string) string {
	switch reason {
	case "", "all":
		return ""
	case "direct", "recommend", "3":
		return LedgerDirect
	case "match", "4":
		return LedgerMatch
	case "manage", "community_base", "5":
		return LedgerManage
	case "admin_adjust":
		return LedgerAdminAdjust
	case "withdraw", "extract":
		return LedgerWithdraw
	case "withdraw_freeze":
		return LedgerFreeze
	case "withdraw_unfreeze":
		return LedgerUnfreeze
	default:
		return reason
	}
}

// LedgerName 展示名。
func LedgerName(entryType string) string {
	switch entryType {
	case LedgerDirect:
		return "直推奖励"
	case LedgerMatch:
		return "对碰奖励"
	case LedgerManage:
		return "三代管理奖"
	case LedgerAdminAdjust:
		return "管理端调整"
	case LedgerWithdraw:
		return "提现"
	case LedgerFreeze:
		return "提现冻结"
	case LedgerUnfreeze:
		return "提现解冻"
	default:
		return entryType
	}
}

// LedgerReason 对外 reason 字段。
func LedgerReason(entryType string) string {
	switch entryType {
	case LedgerDirect:
		return "direct"
	case LedgerMatch:
		return "match"
	case LedgerManage:
		return "manage"
	case LedgerAdminAdjust:
		return "admin_adjust"
	case LedgerWithdraw:
		return "withdraw"
	case LedgerFreeze:
		return "withdraw_freeze"
	case LedgerUnfreeze:
		return "withdraw_unfreeze"
	default:
		return entryType
	}
}

func amountStr(d decimal.Decimal) string {
	return money.Round(d).StringFixed(8)
}

func toRewardItem(e *LedgerEntry) *RewardItem {
	return &RewardItem{
		ID:        e.ID,
		Amount:    amountStr(e.Amount),
		Name:      LedgerName(e.EntryType),
		Reason:    LedgerReason(e.EntryType),
		Address:   "",
		Num:       "",
		Remark:    e.Remark,
		CreatedAt: e.CreatedAt,
	}
}

func paginateRewards(items []*RewardItem, page, pageSize int) []*RewardItem {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultRewardPageSize
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []*RewardItem{}
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

func filterUserRewards(items []*LedgerEntry, want string) []*RewardItem {
	out := make([]*RewardItem, 0, len(items))
	for _, e := range items {
		if want != "" {
			if e.EntryType != want {
				continue
			}
		} else if !IsRewardEntry(e.EntryType) {
			continue
		}
		out = append(out, toRewardItem(e))
	}
	return out
}

// ListUserRewards 用户端 reward_list：默认三类收益；reqType 映射单类；未知码空列表。
func (uc *LedgerUseCase) ListUserRewards(ctx context.Context, userID uint64, reqType string, page int) (*RewardPage, error) {
	want, ok := ReqTypeToEntryType(reqType)
	if !ok {
		return &RewardPage{Items: []*RewardItem{}, Total: 0}, nil
	}
	now := uc.now()
	from := now.AddDate(-1, 0, 0)
	to := now.AddDate(0, 0, 1)
	rows, err := uc.ledger.ListByUser(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	filtered := filterUserRewards(rows, want)
	return &RewardPage{
		Items: paginateRewards(filtered, page, DefaultRewardPageSize),
		Total: len(filtered),
	}, nil
}

// ListAdminRewards 管理端流水：分页 + address + reason；不过滤类型（reason 空则全类型）。
func (uc *LedgerUseCase) ListAdminRewards(ctx context.Context, address, reason string, page int) (*RewardPage, error) {
	entryType := ReasonToEntryType(reason)
	rows, total, err := uc.ledger.ListPaged(ctx, address, entryType, page, DefaultRewardPageSize)
	if err != nil {
		return nil, err
	}
	items := make([]*RewardItem, 0, len(rows))
	for _, e := range rows {
		items = append(items, toRewardItem(e))
	}
	return &RewardPage{Items: items, Total: total}, nil
}
