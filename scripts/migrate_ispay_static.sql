-- 已有库增量：ispay 余额、订单释放档、日结静态计数（可重复执行）
SET @db := DATABASE();

SET @sql := (
    SELECT IF(COUNT(*) = 0,
        'ALTER TABLE users ADD COLUMN ispay_balance DECIMAL(36, 8) NOT NULL DEFAULT 0 COMMENT ''ispay coin balance'' AFTER frozen_balance',
        'SELECT 1')
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'users' AND COLUMN_NAME = 'ispay_balance'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := (
    SELECT IF(COUNT(*) = 0,
        'ALTER TABLE orders ADD COLUMN release_days INT NOT NULL DEFAULT 0 COMMENT ''300|600|750 static release'' AFTER log_index',
        'SELECT 1')
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'orders' AND COLUMN_NAME = 'release_days'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := (
    SELECT IF(COUNT(*) = 0,
        'ALTER TABLE settle_runs ADD COLUMN static_count INT NOT NULL DEFAULT 0 AFTER manage_count',
        'SELECT 1')
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'settle_runs' AND COLUMN_NAME = 'static_count'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
