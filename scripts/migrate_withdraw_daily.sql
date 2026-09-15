-- 单笔提现上限（0 表示不限制；USDT / ISPAY 分开；键名沿用 withdraw_daily_limit）
INSERT INTO business_configs (config_key, name, value, sort_order) VALUES
('withdraw_daily_limit', 'USDT 单笔提现上限', '1000', 44),
('withdraw_daily_limit_ispay', 'ISPAY 单笔提现上限', '1000', 45)
ON DUPLICATE KEY UPDATE name = VALUES(name), sort_order = VALUES(sort_order);
