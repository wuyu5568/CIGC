-- 已有库增量：商品不再按金额+天数唯一（可重复执行）
SET @db := DATABASE();

SET @sql := (
    SELECT IF(COUNT(*) > 0,
        'ALTER TABLE packages DROP INDEX uk_packages_amount_days',
        'SELECT 1')
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'packages' AND INDEX_NAME = 'uk_packages_amount_days'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 下架与 300 天同金额同名的 600/750 复制品，避免用户端看到三份相同商品
UPDATE packages p
JOIN packages src
  ON src.amount = p.amount AND src.title = p.title AND src.release_days = 300
SET p.enabled = 0
WHERE p.release_days IN (600, 750) AND p.enabled = 1;
