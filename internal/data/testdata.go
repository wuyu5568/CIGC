package data

import (
	"context"

	"github.com/cigc/app/internal/biz"
	"gorm.io/gorm"
)

type testDataRepo struct{ data *Data }

// NewTestDataRepo 清测试业务数据，保留用户与邀请/安置。
func NewTestDataRepo(d *Data) biz.TestDataRepo { return &testDataRepo{data: d} }

func (r *testDataRepo) ClearKeepUsers(ctx context.Context) (*biz.TestDataClearResult, error) {
	out := &biz.TestDataClearResult{}
	err := r.data.InTx(ctx, func(ctx context.Context) error {
		db := r.data.Session(ctx)
		if err := db.Exec("SET FOREIGN_KEY_CHECKS = 0").Error; err != nil {
			return err
		}
		var users int64
		if err := db.Model(&UserModel{}).Count(&users).Error; err != nil {
			return err
		}
		out.UsersKept = int(users)

		n, err := execDelete(db, "DELETE FROM match_order_applied")
		if err != nil {
			return err
		}
		_ = n
		if _, err := execDelete(db, "DELETE FROM chain_deposits"); err != nil {
			return err
		}
		if out.LedgerCleared, err = execDelete(db, "DELETE FROM ledger_entries"); err != nil {
			return err
		}
		if out.HoldsCleared, err = execDelete(db, "DELETE FROM cap_overflow_holds"); err != nil {
			return err
		}
		if _, err := execDelete(db, "DELETE FROM user_daily_dynamic"); err != nil {
			return err
		}
		if out.WithdrawsCleared, err = execDelete(db, "DELETE FROM withdraws"); err != nil {
			return err
		}
		if out.OrdersCleared, err = execDelete(db, "DELETE FROM orders"); err != nil {
			return err
		}
		if _, err := execDelete(db, "DELETE FROM user_match_balances"); err != nil {
			return err
		}
		if _, err := execDelete(db, "DELETE FROM login_challenges"); err != nil {
			return err
		}
		if _, err := execDelete(db, "DELETE FROM settle_runs"); err != nil {
			return err
		}
		if err := db.Exec(`
UPDATE users SET
    available_balance = 0,
    recharge_balance = 0,
    frozen_balance = 0,
    frozen_ispay = 0,
    ispay_balance = 0,
    lock_balance = 0,
    lock_ispay = 0,
    paid_amount = 0,
    cap_effective = 0,
    disabled_at = NULL
`).Error; err != nil {
			return err
		}
		return db.Exec("SET FOREIGN_KEY_CHECKS = 1").Error
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func execDelete(db *gorm.DB, sql string) (int, error) {
	res := db.Exec(sql)
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
}
