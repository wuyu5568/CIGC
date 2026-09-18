-- 已有库增量：规格图（可重复执行）
SET @db := DATABASE();

SET @sql := (
    SELECT IF(COUNT(*) = 0,
        'ALTER TABLE package_skus ADD COLUMN image VARCHAR(512) NOT NULL DEFAULT '''' AFTER amount',
        'SELECT 1')
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'package_skus' AND COLUMN_NAME = 'image'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
