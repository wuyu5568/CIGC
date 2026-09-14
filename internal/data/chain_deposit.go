package data

import (
	"context"
	"errors"

	"github.com/cigc/app/internal/biz"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type chainCursorRepo struct{ data *Data }

// NewChainCursorRepo 扫块游标。
func NewChainCursorRepo(d *Data) biz.ChainCursorRepo { return &chainCursorRepo{data: d} }

func (r *chainCursorRepo) Get(ctx context.Context, name string) (uint64, error) {
	var m ChainScanCursorModel
	err := r.data.Session(ctx).Where("name = ?", name).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return m.BlockNumber, nil
}

func (r *chainCursorRepo) Set(ctx context.Context, name string, block uint64) error {
	m := ChainScanCursorModel{Name: name, BlockNumber: block}
	return r.data.Session(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"block_number"}),
	}).Create(&m).Error
}

func (r *chainCursorRepo) Advance(ctx context.Context, name string, n uint64) error {
	return r.data.Session(ctx).Exec(
		"INSERT INTO chain_scan_cursors (name, block_number) VALUES (?, ?) ON DUPLICATE KEY UPDATE block_number = IF(? > block_number, ?, block_number)",
		name, n, n, n,
	).Error
}

type chainDepositRepo struct{ data *Data }

// NewChainDepositRepo 链上入账审计。
func NewChainDepositRepo(d *Data) biz.ChainDepositRepo { return &chainDepositRepo{data: d} }

func (r *chainDepositRepo) FindByEvent(ctx context.Context, txHash string, logIndex int) (*biz.ChainDeposit, error) {
	var m ChainDepositModel
	err := r.data.Session(ctx).Where("tx_hash = ? AND log_index = ?", txHash, logIndex).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toBizDeposit(&m), nil
}

func toBizDeposit(m *ChainDepositModel) *biz.ChainDeposit {
	if m == nil {
		return nil
	}
	return &biz.ChainDeposit{
		ID: m.ID, TxHash: m.TxHash, LogIndex: m.LogIndex, FromAddr: m.FromAddr, ToAddr: m.ToAddr,
		Amount: m.Amount, BlockNumber: m.BlockNumber, Status: m.Status, OrderID: m.OrderID,
		Remark: m.Remark, CreatedAt: m.CreatedAt,
	}
}

func (r *chainDepositRepo) ListByOrder(ctx context.Context, orderID uint64) ([]*biz.ChainDeposit, error) {
	var rows []ChainDepositModel
	if err := r.data.Session(ctx).
		Where("order_id = ? AND status = ?", orderID, biz.DepositMatched).
		Order("id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.ChainDeposit, len(rows))
	for i := range rows {
		out[i] = toBizDeposit(&rows[i])
	}
	return out, nil
}

func (r *chainDepositRepo) ListByFrom(ctx context.Context, fromAddr string, page, pageSize int) ([]*biz.ChainDeposit, int, error) {
	return r.listDeposits(ctx, fromAddr, false, page, pageSize)
}

func (r *chainDepositRepo) ListAdmin(ctx context.Context, fromAddr string, page, pageSize int) ([]*biz.ChainDeposit, int, error) {
	return r.listDeposits(ctx, fromAddr, true, page, pageSize)
}

func (r *chainDepositRepo) listDeposits(ctx context.Context, fromAddr string, fuzzy bool, page, pageSize int) ([]*biz.ChainDeposit, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = biz.DefaultRewardPageSize
	}
	base := r.data.Session(ctx).Model(&ChainDepositModel{})
	if fromAddr != "" {
		if fuzzy {
			base = base.Where("from_addr LIKE ?", "%"+fromAddr+"%")
		} else {
			base = base.Where("from_addr = ?", fromAddr)
		}
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []ChainDepositModel
	offset := (page - 1) * pageSize
	if err := base.Session(&gorm.Session{}).Order("id DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.ChainDeposit, len(rows))
	for i := range rows {
		out[i] = toBizDeposit(&rows[i])
	}
	return out, int(total), nil
}

func (r *chainDepositRepo) Create(ctx context.Context, d *biz.ChainDeposit) error {
	m := ChainDepositModel{
		TxHash: d.TxHash, LogIndex: d.LogIndex, FromAddr: d.FromAddr, ToAddr: d.ToAddr,
		Amount: d.Amount, BlockNumber: d.BlockNumber, Status: d.Status, OrderID: d.OrderID, Remark: d.Remark,
	}
	err := r.data.Session(ctx).Create(&m).Error
	if err != nil {
		if isDuplicateKey(err) {
			return biz.ErrOrderConflict
		}
		return err
	}
	d.ID = m.ID
	return nil
}
