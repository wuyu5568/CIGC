package data

import (
	"context"
	"errors"
	"strings"

	"github.com/cigc/app/internal/biz"
	"gorm.io/gorm"
)

type placementRepo struct{ data *Data }

// NewPlacementRepo 安置仓储。
func NewPlacementRepo(d *Data) biz.PlacementRepo { return &placementRepo{data: d} }

func toBizPlacement(m *UserPlacementModel) *biz.Placement {
	return &biz.Placement{
		ID:        m.ID,
		UserID:    m.UserID,
		SponsorID: m.SponsorID,
		Side:      m.Side,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func (r *placementRepo) Create(ctx context.Context, p *biz.Placement) (*biz.Placement, error) {
	m := UserPlacementModel{
		UserID:    p.UserID,
		SponsorID: p.SponsorID,
		Side:      p.Side,
	}
	err := r.data.Session(ctx).Create(&m).Error
	if err != nil {
		if isDuplicateKey(err) {
			return nil, biz.ErrPlacementConflict
		}
		return nil, err
	}
	return r.FindByUserID(ctx, m.UserID)
}

func (r *placementRepo) FindByUserID(ctx context.Context, userID uint64) (*biz.Placement, error) {
	var m UserPlacementModel
	err := r.data.Session(ctx).Where("user_id = ?", userID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toBizPlacement(&m), nil
}

func (r *placementRepo) FindChild(ctx context.Context, sponsorID uint64, side string) (*biz.Placement, error) {
	var m UserPlacementModel
	err := r.data.Session(ctx).Where("sponsor_id = ? AND side = ?", sponsorID, side).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toBizPlacement(&m), nil
}

func (r *placementRepo) ListBySponsor(ctx context.Context, sponsorID uint64) ([]*biz.Placement, error) {
	var rows []UserPlacementModel
	if err := r.data.Session(ctx).Where("sponsor_id = ?", sponsorID).
		Order("side ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.Placement, len(rows))
	for i := range rows {
		out[i] = toBizPlacement(&rows[i])
	}
	return out, nil
}

func (r *placementRepo) ListPaged(ctx context.Context, address string, page, pageSize int) ([]*biz.PlacementView, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = biz.DefaultRewardPageSize
	}
	base := r.data.Session(ctx).Table("user_placements").
		Joins("LEFT JOIN users u ON u.id = user_placements.user_id").
		Joins("LEFT JOIN users s ON s.id = user_placements.sponsor_id")
	address = strings.TrimSpace(address)
	if address != "" {
		like := "%" + address + "%"
		base = base.Where("u.address LIKE ? OR s.address LIKE ?", like, like)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type row struct {
		UserPlacementModel
		UserAddress    string `gorm:"column:user_address"`
		SponsorAddress string `gorm:"column:sponsor_address"`
	}
	var rows []row
	offset := (page - 1) * pageSize
	if err := base.Session(&gorm.Session{}).
		Select("user_placements.*, u.address AS user_address, s.address AS sponsor_address").
		Order("user_placements.id DESC").
		Offset(offset).Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.PlacementView, len(rows))
	for i := range rows {
		p := toBizPlacement(&rows[i].UserPlacementModel)
		out[i] = &biz.PlacementView{
			Placement:      *p,
			UserAddress:    rows[i].UserAddress,
			SponsorAddress: rows[i].SponsorAddress,
		}
	}
	return out, int(total), nil
}

func (r *placementRepo) ListAll(ctx context.Context) ([]*biz.Placement, error) {
	var rows []UserPlacementModel
	if err := r.data.Session(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.Placement, len(rows))
	for i := range rows {
		out[i] = toBizPlacement(&rows[i])
	}
	return out, nil
}
