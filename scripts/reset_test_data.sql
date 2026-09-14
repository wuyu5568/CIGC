-- 清测试业务数据，只留创世用户。商品档位与业务配置不动。
-- 由 scripts/reset_test_data.sh 注入 @genesis_address 后执行。

SET NAMES utf8mb4;
SET @genesis_id := (
    SELECT id FROM users
    WHERE address = @genesis_address
    LIMIT 1
);

START TRANSACTION;

SET FOREIGN_KEY_CHECKS = 0;

DELETE FROM match_order_applied;
DELETE FROM chain_deposits;
DELETE FROM ledger_entries;
DELETE FROM cap_overflow_holds;
DELETE FROM user_daily_dynamic;
DELETE FROM withdraws;
DELETE FROM orders;
DELETE FROM user_placements;
DELETE FROM user_match_balances;
DELETE FROM user_recommends;
DELETE FROM login_challenges;
DELETE FROM settle_runs;

DELETE FROM users WHERE @genesis_id IS NULL OR id <> @genesis_id;

SET FOREIGN_KEY_CHECKS = 1;

UPDATE users
SET
    inviter_id = NULL,
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
WHERE @genesis_id IS NOT NULL AND id = @genesis_id;

INSERT INTO user_recommends (user_id, path)
SELECT id, '' FROM users WHERE @genesis_id IS NOT NULL AND id = @genesis_id
ON DUPLICATE KEY UPDATE path = '';

COMMIT;

SELECT
    (SELECT COUNT(*) FROM users) AS users_left,
    (SELECT COUNT(*) FROM orders) AS orders_left,
    (SELECT COUNT(*) FROM ledger_entries) AS ledger_left,
    (SELECT COUNT(*) FROM withdraws) AS withdraws_left,
    (SELECT COUNT(*) FROM chain_deposits) AS deposits_left,
    (SELECT COUNT(*) FROM user_placements) AS placements_left,
    (SELECT COUNT(*) FROM settle_runs) AS settle_left,
    (SELECT COUNT(*) FROM packages) AS packages_kept,
    (SELECT COUNT(*) FROM business_configs) AS configs_kept,
    CASE WHEN @genesis_id IS NULL THEN 'missing' ELSE 'kept' END AS genesis_user;
