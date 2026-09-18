package data

import (
	"context"
	"errors"

	"github.com/cigc/app/internal/biz"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type shippingRepo struct{ data *Data }

func NewShippingAddressRepo(d *Data) biz.ShippingAddressRepo { return &shippingRepo{data: d} }

func toBizShipping(m *ShippingAddressModel) *biz.ShippingAddress {
	if m == nil {
		return nil
	}
	return &biz.ShippingAddress{
		ID:        m.ID,
		UserID:    m.UserID,
		Name:      m.Name,
		Contact:   m.Contact,
		Address:   m.Address,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func (r *shippingRepo) FindByUserID(ctx context.Context, userID uint64) (*biz.ShippingAddress, error) {
	var m ShippingAddressModel
	if err := r.data.Session(ctx).Where("user_id = ?", userID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || isMissingTable(err) {
			return nil, biz.ErrNotFound
		}
		return nil, err
	}
	return toBizShipping(&m), nil
}

func (r *shippingRepo) Upsert(ctx context.Context, a *biz.ShippingAddress) (*biz.ShippingAddress, error) {
	if a == nil || a.UserID == 0 {
		return nil, biz.ErrShippingInvalid
	}
	m := ShippingAddressModel{
		UserID:  a.UserID,
		Name:    a.Name,
		Contact: a.Contact,
		Address: a.Address,
	}
	err := r.data.Session(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "contact", "address"}),
	}).Create(&m).Error
	if err != nil {
		return nil, err
	}
	return r.FindByUserID(ctx, a.UserID)
}
