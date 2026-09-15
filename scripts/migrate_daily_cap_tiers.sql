-- 日封顶档位表（JSON）。已有值不覆盖。
INSERT INTO business_configs (config_key, name, value, sort_order) VALUES
('daily_cap_tiers', '日封顶档位', '[{"max_amount":"3000","daily_cap":"600"},{"max_amount":"6000","daily_cap":"1800"},{"max_amount":"12000","daily_cap":"4000"},{"max_amount":"24000","daily_cap":"16000"},{"max_amount":"36000","daily_cap":"24000"},{"max_amount":"50000","daily_cap":"30000"},{"max_amount":"70000","daily_cap":"42000"},{"max_amount":"100000","daily_cap":"60000"},{"max_amount":"","daily_cap":"100000"}]', 80)
ON DUPLICATE KEY UPDATE name = VALUES(name), sort_order = VALUES(sort_order);
