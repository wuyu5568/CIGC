-- USDT 提现手续费比例（0–1，管理端可改；ISPAY 本刀 0）
INSERT INTO business_configs (config_key, name, value, sort_order) VALUES
('withdraw_fee_rate', 'USDT提现手续费比例', '0.10', 45)
ON DUPLICATE KEY UPDATE name = VALUES(name), sort_order = VALUES(sort_order);
