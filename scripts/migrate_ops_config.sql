-- 运营可改：管理奖代数、ISPAY 提现门槛/手续费、冻结清除时间（小时）。已有值不覆盖。
INSERT INTO business_configs (config_key, name, value, sort_order)
SELECT 'overflow_clear_hours', '冻结清除时间（小时）',
  CAST(GREATEST(1, LEAST(720, (CAST(value AS SIGNED) - 1) * 24)) AS CHAR),
  60
FROM business_configs WHERE config_key = 'overflow_clear_days'
ON DUPLICATE KEY UPDATE name = VALUES(name), sort_order = VALUES(sort_order);

INSERT INTO business_configs (config_key, name, value, sort_order) VALUES
('manage_generations', '管理奖代数', '3', 31),
('min_withdraw_amount_ispay', 'ISPAY最低提现金额', '0', 41),
('withdraw_fee_rate_ispay', 'ISPAY提现手续费比例', '0', 48),
('overflow_clear_hours', '冻结清除时间（小时）', '72', 60)
ON DUPLICATE KEY UPDATE name = VALUES(name), sort_order = VALUES(sort_order);

DELETE FROM business_configs WHERE config_key = 'overflow_clear_days';

UPDATE business_configs SET name = '管理奖励总池比例' WHERE config_key = 'manage_rate';
UPDATE business_configs SET name = 'USDT最低提现金额' WHERE config_key = 'min_withdraw_amount';
