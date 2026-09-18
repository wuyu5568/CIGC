package biz

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	shippingNameMax    = 64
	shippingContactMax = 64
	shippingAddressMax = 512
)

// ShippingAddress 用户唯一收货地址。
type ShippingAddress struct {
	ID        uint64
	UserID    uint64
	Name      string
	Contact   string
	Address   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Complete 姓名、联系方式、地址均已填写。
func (a *ShippingAddress) Complete() bool {
	return a != nil &&
		strings.TrimSpace(a.Name) != "" &&
		strings.TrimSpace(a.Contact) != "" &&
		strings.TrimSpace(a.Address) != ""
}

// ShippingAddressRepo 每人一行收货地址。
type ShippingAddressRepo interface {
	FindByUserID(ctx context.Context, userID uint64) (*ShippingAddress, error)
	Upsert(ctx context.Context, a *ShippingAddress) (*ShippingAddress, error)
}

// NormalizeShippingAddress 去掉首尾空白并截断到字段上限。
func NormalizeShippingAddress(userID uint64, name, contact, address string) *ShippingAddress {
	return &ShippingAddress{
		UserID:  userID,
		Name:    clipRunes(name, shippingNameMax),
		Contact: clipRunes(contact, shippingContactMax),
		Address: clipRunes(address, shippingAddressMax),
	}
}

// SetShipping 注入收货地址仓储；测试可不设。
func (uc *UserUseCase) SetShipping(repo ShippingAddressRepo) {
	if uc == nil {
		return
	}
	uc.shipping = repo
}

// GetShippingAddress 没有记录时返回 nil。
func (uc *UserUseCase) GetShippingAddress(ctx context.Context, userID uint64) (*ShippingAddress, error) {
	if uc == nil || uc.shipping == nil || userID == 0 {
		return nil, nil
	}
	a, err := uc.shipping.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return a, nil
}

// RequireShippingAddress 结算前校验：必须已填写完整收货地址。
func (uc *UserUseCase) RequireShippingAddress(ctx context.Context, userID uint64) (*ShippingAddress, error) {
	a, err := uc.GetShippingAddress(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !a.Complete() {
		return nil, ErrShippingRequired
	}
	return a, nil
}

// SaveShippingAddress 新增或覆盖当前用户唯一收货地址。
func (uc *UserUseCase) SaveShippingAddress(ctx context.Context, userID uint64, name, contact, address string) (*ShippingAddress, error) {
	if uc == nil || uc.shipping == nil {
		return nil, ErrShippingInvalid
	}
	if _, err := uc.GetProfile(ctx, userID); err != nil {
		return nil, err
	}
	a := NormalizeShippingAddress(userID, name, contact, address)
	if !a.Complete() {
		return nil, ErrShippingInvalid
	}
	return uc.shipping.Upsert(ctx, a)
}
