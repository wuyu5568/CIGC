package biz

import (
	"context"
	"time"

	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
)

const maxMatchAncestorWalk = 4096

// MatchBalance 是会员左右区待对碰结余。
type MatchBalance struct {
	UserID      uint64
	LeftRemain  decimal.Decimal
	RightRemain decimal.Decimal
}

// MatchRepo 对碰结余与「订单业绩已累加」防重。
type MatchRepo interface {
	Get(ctx context.Context, userID uint64) (*MatchBalance, error)
	AddRemain(ctx context.Context, userID uint64, side string, delta decimal.Decimal) error
	SaveRemains(ctx context.Context, userID uint64, left, right decimal.Decimal) error
	ListAll(ctx context.Context) ([]*MatchBalance, error)
	TryApplyOrder(ctx context.Context, orderID uint64, settleDate time.Time) (applied bool, err error)
}

// MatchPair 用当前结余配对：credit=pair×rate，两边各减 pair。日封顶在入账时按当日累计处理。
func MatchPair(left, right, rate decimal.Decimal) (credit, pair, newLeft, newRight decimal.Decimal) {
	left = money.Round(left)
	right = money.Round(right)
	rate = money.Round(rate)
	if left.LessThan(right) {
		pair = left
	} else {
		pair = right
	}
	if !pair.IsPositive() {
		return decimal.Zero, decimal.Zero, left, right
	}
	credit = money.Round(pair.Mul(rate))
	if credit.IsNegative() {
		credit = decimal.Zero
	}
	return credit, pair, money.Round(left.Sub(pair)), money.Round(right.Sub(pair))
}
