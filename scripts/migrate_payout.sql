-- 提现打款元数据（已有库可能缺 payout_error）
SET @db := DATABASE();

SET @sql := (
    SELECT IF(COUNT(*) = 0,
        'ALTER TABLE withdraws ADD COLUMN payout_error VARCHAR(255) NOT NULL DEFAULT '''' COMMENT ''last payout error'' AFTER tx_hash',
        'SELECT 1')
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'withdraws' AND COLUMN_NAME = 'payout_error'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
