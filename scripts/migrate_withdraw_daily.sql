-- 每日提现上限（上海自然日；0 表示不限制；USDT / ISPAY 分开计）
INSERT INTO business_configs (config_key, name, value, sort_order) VALUES
('withdraw_daily_limit', 'USDT每日提现上限', '1000', 46),
('withdraw_daily_limit_ispay', 'ISPAY每日提现上限', '1000', 47)
ON DUPLICATE KEY UPDATE name = VALUES(name), sort_order = VALUES(sort_order);
