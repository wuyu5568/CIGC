-- 已有库增量：套餐默认释放天数（可重复执行）
SET @db := DATABASE();

SET @sql := (
    SELECT IF(COUNT(*) = 0,
        'ALTER TABLE packages ADD COLUMN release_days INT NOT NULL DEFAULT 300 COMMENT ''300|600|750 default static release'' AFTER daily_cap',
        'SELECT 1')
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'packages' AND COLUMN_NAME = 'release_days'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
