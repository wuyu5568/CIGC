-- 充值页入账、仅用于买套餐的可用余额（与可提现 available_balance 分离）
SET @db := DATABASE();

SET @sql := (
    SELECT IF(COUNT(*) = 0,
        'ALTER TABLE users ADD COLUMN recharge_balance DECIMAL(36, 8) NOT NULL DEFAULT 0 COMMENT ''buy-only USDT from recharge page'' AFTER available_balance',
        'SELECT 1')
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'users' AND COLUMN_NAME = 'recharge_balance'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
