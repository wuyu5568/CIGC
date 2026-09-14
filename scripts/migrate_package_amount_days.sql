-- 已有库增量：金额唯一改为「金额 + 释放天数」，并把 300 天商品复制到 600/750（可重复执行）
SET @db := DATABASE();

SET @sql := (
    SELECT IF(COUNT(*) > 0,
        'ALTER TABLE packages DROP INDEX uk_packages_amount',
        'SELECT 1')
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'packages' AND INDEX_NAME = 'uk_packages_amount'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := (
    SELECT IF(COUNT(*) = 0,
        'ALTER TABLE packages ADD UNIQUE KEY uk_packages_amount_days (amount, release_days)',
        'SELECT 1')
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'packages' AND INDEX_NAME = 'uk_packages_amount_days'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

INSERT IGNORE INTO packages (amount, title, goods_desc, daily_cap, release_days, sort_order, enabled, image)
SELECT amount, title, goods_desc, daily_cap, 600, sort_order, enabled, image
FROM (
    SELECT amount, title, goods_desc, daily_cap, sort_order, enabled, image
    FROM packages
    WHERE release_days = 300
) src;

INSERT IGNORE INTO packages (amount, title, goods_desc, daily_cap, release_days, sort_order, enabled, image)
SELECT amount, title, goods_desc, daily_cap, 750, sort_order, enabled, image
FROM (
    SELECT amount, title, goods_desc, daily_cap, sort_order, enabled, image
    FROM packages
    WHERE release_days = 300
) src;
