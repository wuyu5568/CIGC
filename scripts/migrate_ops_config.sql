-- 运营可改：管理奖代数、ISPAY 提现门槛/手续费、冻结清除时间（小时）。已有值不覆盖。
INSERT INTO business_configs (config_key, name, value, sort_order)
SELECT 'overflow_clear_hours', '冻结清除时间（小时）',
  CAST(GREATEST(1, LEAST(720, (CAST(value AS SIGNED) - 1) * 24)) AS CHAR),
  60
FROM business_configs WHERE config_key = 'overflow_clear_days'
ON DUPLICATE KEY UPDATE name = VALUES(name), sort_order = VALUES(sort_order);

INSERT INTO business_configs (config_key, name, value, sort_order) VALUES
('manage_generations', '管理奖代数', '3', 31),
('min_withdraw_amount_ispay', 'ISPAY 单笔最低提现', '0', 41),
('withdraw_fee_rate_ispay', 'ISPAY 提现手续费', '0', 43),
('overflow_clear_hours', '冻结清除时间', '72', 60),
('withdraw_enabled', '提现开关', '1', 70)
ON DUPLICATE KEY UPDATE name = VALUES(name), sort_order = VALUES(sort_order);

DELETE FROM business_configs WHERE config_key = 'overflow_clear_days';

UPDATE business_configs SET name = '直推奖比例', sort_order = 10 WHERE config_key = 'direct_rate';
UPDATE business_configs SET name = '对碰奖比例', sort_order = 20 WHERE config_key = 'match_rate';
UPDATE business_configs SET name = '管理奖比例', sort_order = 30 WHERE config_key = 'manage_rate';
UPDATE business_configs SET name = '管理奖代数', sort_order = 31 WHERE config_key = 'manage_generations';
UPDATE business_configs SET name = 'USDT 单笔最低提现', sort_order = 40 WHERE config_key = 'min_withdraw_amount';
UPDATE business_configs SET name = 'ISPAY 单笔最低提现', sort_order = 41 WHERE config_key = 'min_withdraw_amount_ispay';
UPDATE business_configs SET name = 'USDT 提现手续费', sort_order = 42 WHERE config_key = 'withdraw_fee_rate';
UPDATE business_configs SET name = 'ISPAY 提现手续费', sort_order = 43 WHERE config_key = 'withdraw_fee_rate_ispay';
UPDATE business_configs SET name = 'USDT 单笔提现上限', sort_order = 44 WHERE config_key = 'withdraw_daily_limit';
UPDATE business_configs SET name = 'ISPAY 单笔提现上限', sort_order = 45 WHERE config_key = 'withdraw_daily_limit_ispay';
UPDATE business_configs SET name = 'ISPAY 测试现价', sort_order = 50 WHERE config_key = 'ispay_price';
UPDATE business_configs SET name = '冻结清除时间', sort_order = 60 WHERE config_key = 'overflow_clear_hours';
UPDATE business_configs SET name = '提现开关', sort_order = 70 WHERE config_key = 'withdraw_enabled';
