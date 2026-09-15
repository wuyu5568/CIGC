-- 已有库增量：Web3 商品富文本详情（可重复执行）
SET @db := DATABASE();

SET @sql := (
    SELECT IF(COUNT(*) = 0,
        'ALTER TABLE packages ADD COLUMN detail MEDIUMTEXT NULL COMMENT ''web3 goods rich text detail'' AFTER image',
        'SELECT 1')
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'packages' AND COLUMN_NAME = 'detail'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
