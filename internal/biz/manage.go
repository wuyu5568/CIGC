package biz

import (
	"context"
	"fmt"

	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
)

const maxManageAncestorWalk = 64

// ManageShares 把管理奖总池均分 n 代；余数贴最近一代（下标 0）。n<1 按 3。
func ManageShares(pool decimal.Decimal, n int) []decimal.Decimal {
	if n < 1 {
		n = defaultManageGens
	}
	if n > maxManageGens {
		n = maxManageGens
	}
	pool = money.Round(pool)
	out := make([]decimal.Decimal, n)
	if !pool.IsPositive() {
		return out
	}
	part := money.Round(pool.Div(decimal.NewFromInt(int64(n))))
	rest := pool
	for i := n - 1; i >= 1; i-- {
		out[i] = part
		rest = money.Round(rest.Sub(part))
	}
	out[0] = rest
	return out
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
