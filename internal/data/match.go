package data

import (
	"context"
	"errors"
	"time"

	"github.com/cigc/app/internal/biz"
	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type matchRepo struct{ data *Data }

// NewMatchRepo 对碰结余仓储。
func NewMatchRepo(d *Data) biz.MatchRepo { return &matchRepo{data: d} }

func (r *matchRepo) Get(ctx context.Context, userID uint64) (*biz.MatchBalance, error) {
	var m UserMatchBalanceModel
	err := r.data.Session(ctx).Where("user_id = ?", userID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &biz.MatchBalance{UserID: userID}, nil
	}
	if err != nil {
		return nil, err
	}
	return &biz.MatchBalance{
		UserID:      m.UserID,
		LeftRemain:  m.LeftRemain,
		RightRemain: m.RightRemain,
	}, nil
}

func (r *matchRepo) AddRemain(ctx context.Context, userID uint64, side string, delta decimal.Decimal) error {
	delta = money.Round(delta)
	if userID == 0 || !delta.IsPositive() {
		return nil
	}
	left, right := decimal.Zero, decimal.Zero
	switch side {
	case biz.SideLeft:
		left = delta
	case biz.SideRight:
		right = delta
	default:
		return biz.ErrPlacementInvalidSide
	}
	m := UserMatchBalanceModel{UserID: userID, LeftRemain: left, RightRemain: right}
	return r.data.Session(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"left_remain":  gorm.Expr("left_remain + ?", left),
			"right_remain": gorm.Expr("right_remain + ?", right),
		}),
	}).Create(&m).Error
}

func (r *matchRepo) SaveRemains(ctx context.Context, userID uint64, left, right decimal.Decimal) error {
	left = money.Round(left)
	right = money.Round(right)
	m := UserMatchBalanceModel{UserID: userID, LeftRemain: left, RightRemain: right}
	return r.data.Session(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"left_remain":  left,
			"right_remain": right,
		}),
	}).Create(&m).Error
}

func (r *matchRepo) ListAll(ctx context.Context) ([]*biz.MatchBalance, error) {
	var rows []UserMatchBalanceModel
	if err := r.data.Session(ctx).Order("user_id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.MatchBalance, len(rows))
	for i := range rows {
		out[i] = &biz.MatchBalance{
			UserID:      rows[i].UserID,
			LeftRemain:  rows[i].LeftRemain,
			RightRemain: rows[i].RightRemain,
		}
	}
	return out, nil
}

func (r *matchRepo) TryApplyOrder(ctx context.Context, orderID uint64, settleDate time.Time) (bool, error) {
	m := MatchOrderAppliedModel{OrderID: orderID, SettleDate: settleDate}
	err := r.data.Session(ctx).Create(&m).Error
	if err != nil {
		if isDuplicateKey(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
