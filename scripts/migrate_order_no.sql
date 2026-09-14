-- 认购订单编号（可重复执行）
SET @db := DATABASE();

SET @sql := (
    SELECT IF(COUNT(*) = 0,
        'ALTER TABLE orders ADD COLUMN order_no VARCHAR(32) NULL COMMENT ''C + 6-digit id'' AFTER id',
        'SELECT 1')
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'orders' AND COLUMN_NAME = 'order_no'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

UPDATE orders
SET order_no = CONCAT('C', LPAD(id, 6, '0'))
WHERE order_no IS NULL OR order_no = '';

SET @sql := (
    SELECT IF(COUNT(*) = 0,
        'ALTER TABLE orders ADD UNIQUE KEY uk_orders_no (order_no)',
        'SELECT 1')
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'orders' AND INDEX_NAME = 'uk_orders_no'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
