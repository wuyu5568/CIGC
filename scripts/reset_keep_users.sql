-- 清测试业务数据，保留全部用户地址及邀请/安置。商品档位与业务配置不动。
-- 链上扫块游标不动，避免旧入账事件被重新匹配。

SET NAMES utf8mb4;

SELECT COUNT(*) AS users_before FROM users;
SELECT COUNT(*) AS orders_before FROM orders;
SELECT COUNT(*) AS ledger_before FROM ledger_entries;

START TRANSACTION;

SET FOREIGN_KEY_CHECKS = 0;

DELETE FROM match_order_applied;
DELETE FROM chain_deposits;
DELETE FROM ledger_entries;
DELETE FROM cap_overflow_holds;
DELETE FROM user_daily_dynamic;
DELETE FROM withdraws;
DELETE FROM orders;
DELETE FROM user_match_balances;
DELETE FROM login_challenges;
DELETE FROM settle_runs;

SET FOREIGN_KEY_CHECKS = 1;

UPDATE users
SET
    available_balance = 0,
    recharge_balance = 0,
    frozen_balance = 0,
    frozen_ispay = 0,
    ispay_balance = 0,
    lock_balance = 0,
    lock_ispay = 0,
    paid_amount = 0,
    cap_effective = 0,
    disabled_at = NULL;

COMMIT;

SELECT
    (SELECT COUNT(*) FROM users) AS users_kept,
    (SELECT COUNT(*) FROM user_recommends) AS recommends_kept,
    (SELECT COUNT(*) FROM user_placements) AS placements_kept,
    (SELECT COUNT(*) FROM orders) AS orders_left,
    (SELECT COUNT(*) FROM ledger_entries) AS ledger_left,
    (SELECT COUNT(*) FROM withdraws) AS withdraws_left,
    (SELECT COUNT(*) FROM cap_overflow_holds) AS holds_left,
    (SELECT COUNT(*) FROM chain_deposits) AS deposits_left,
    (SELECT COUNT(*) FROM settle_runs) AS settle_left,
    (SELECT COUNT(*) FROM packages) AS packages_kept,
    (SELECT COUNT(*) FROM business_configs) AS configs_kept;
