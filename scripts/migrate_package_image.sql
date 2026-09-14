-- 已有库增量：Web3 商品图片（可重复执行）
SET @db := DATABASE();

SET @sql := (
    SELECT IF(COUNT(*) = 0,
        'ALTER TABLE packages ADD COLUMN image VARCHAR(512) NOT NULL DEFAULT '''' COMMENT ''goods image url or path'' AFTER enabled',
        'SELECT 1')
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'packages' AND COLUMN_NAME = 'image'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
