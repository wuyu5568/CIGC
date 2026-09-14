-- 已有库增量：测试 ispay 现价（管理端可改）
INSERT INTO business_configs (config_key, name, value, sort_order) VALUES
('ispay_price', '测试 ispay 现价（U）', '2000', 50)
ON DUPLICATE KEY UPDATE name = VALUES(name), sort_order = VALUES(sort_order);
