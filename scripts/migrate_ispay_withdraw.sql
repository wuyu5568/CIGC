-- ISPAY 提现冻结（与收益冻结 lock_ispay、USDT 提现 frozen_balance 分离）
SET @db := DATABASE();

SET @sql := (
    SELECT IF(COUNT(*) = 0,
        'ALTER TABLE users ADD COLUMN frozen_ispay DECIMAL(36, 8) NOT NULL DEFAULT 0 COMMENT ''ispay withdraw freeze'' AFTER frozen_balance',
        'SELECT 1')
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'users' AND COLUMN_NAME = 'frozen_ispay'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := (
    SELECT IF(COUNT(*) = 0,
        'ALTER TABLE withdraws ADD COLUMN asset VARCHAR(16) NOT NULL DEFAULT ''usdt'' COMMENT ''usdt|ispay'' AFTER credited_amount',
        'SELECT 1')
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'withdraws' AND COLUMN_NAME = 'asset'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
