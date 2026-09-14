package biz

import (
	"context"
	"fmt"

	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
)

const maxManageAncestorWalk = 64
const manageAncestorWant = 3

// ManageShares 把管理奖总池均分三代；余数贴最近一代（下标 0）。
func ManageShares(pool decimal.Decimal) [3]decimal.Decimal {
	pool = money.Round(pool)
	if !pool.IsPositive() {
		return [3]decimal.Decimal{}
	}
	third := money.Round(pool.Div(decimal.NewFromInt(3)))
	return [3]decimal.Decimal{
		money.Round(pool.Sub(third).Sub(third)),
		third,
		third,
	}
}

func manageRemark(gen int, sourceUserID uint64, uniq string) string {
	if uniq == "" {
		uniq = "default"
	}
	return fmt.Sprintf("manage gen=%d source=%d key=%s", gen, sourceUserID, uniq)
}

func manageUniqFromMatch(m *LedgerEntry) string {
	if m != nil && m.OrderID != nil && *m.OrderID > 0 {
		return fmt.Sprintf("order=%d", *m.OrderID)
	}
	return "leftover"
}

// FindActivatedRecommendAncestors 沿推荐关系向上找已激活用户，跳过未激活，最多 want 人。
func FindActivatedRecommendAncestors(ctx context.Context, users UserRepo, userID uint64, want int) ([]uint64, error) {
	if users == nil || userID == 0 || want <= 0 {
		return nil, nil
	}
	out := make([]uint64, 0, want)
	cur := userID
	seen := map[uint64]struct{}{}
	for i := 0; i < maxManageAncestorWalk && len(out) < want; i++ {
		if _, ok := seen[cur]; ok {
			break
		}
		seen[cur] = struct{}{}
		u, err := users.FindByID(ctx, cur)
		if err != nil {
			return out, err
		}
		if u.InviterID == nil {
			break
		}
		inv, err := users.FindByID(ctx, *u.InviterID)
		if err != nil {
			return out, err
		}
		if inv.IsActivated() {
			out = append(out, inv.ID)
		}
		cur = inv.ID
	}
	return out, nil
}
