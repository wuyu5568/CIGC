-- 未激活收益冻结（与提现 frozen_balance 分离）
SET @db := DATABASE();

SET @sql := (
    SELECT IF(COUNT(*) = 0,
        'ALTER TABLE users ADD COLUMN lock_balance DECIMAL(36, 8) NOT NULL DEFAULT 0 COMMENT ''inactive reward USDT'' AFTER ispay_balance',
        'SELECT 1')
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'users' AND COLUMN_NAME = 'lock_balance'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := (
    SELECT IF(COUNT(*) = 0,
        'ALTER TABLE users ADD COLUMN lock_ispay DECIMAL(36, 8) NOT NULL DEFAULT 0 COMMENT ''inactive reward ispay'' AFTER lock_balance',
        'SELECT 1')
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'users' AND COLUMN_NAME = 'lock_ispay'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
