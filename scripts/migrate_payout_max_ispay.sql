-- ISPAY 热钱包单笔打款上限。已有值不覆盖。
INSERT INTO business_configs (config_key, name, value, sort_order) VALUES
('payout_max_ispay', 'ISPAY 单笔打款上限', '1000', 46)
ON DUPLICATE KEY UPDATE name = VALUES(name), sort_order = VALUES(sort_order);
