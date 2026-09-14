-- 已有库增量：链上核销游标与入账事件表
CREATE TABLE IF NOT EXISTS chain_scan_cursors (
    name         VARCHAR(64)     NOT NULL PRIMARY KEY,
    block_number BIGINT UNSIGNED NOT NULL DEFAULT 0,
    updated_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

CREATE TABLE IF NOT EXISTS chain_deposits (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    tx_hash      VARCHAR(80)     NOT NULL,
    log_index    INT             NOT NULL,
    from_addr    VARCHAR(64)     NOT NULL,
    to_addr      VARCHAR(64)     NOT NULL,
    amount       DECIMAL(36, 8)  NOT NULL,
    block_number BIGINT UNSIGNED NOT NULL DEFAULT 0,
    status       VARCHAR(16)     NOT NULL COMMENT 'matched|abnormal|skipped',
    order_id     BIGINT UNSIGNED NULL,
    remark       VARCHAR(255)    NOT NULL DEFAULT '',
    created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_chain_deposits_event (tx_hash, log_index),
    KEY idx_chain_deposits_from (from_addr),
    KEY idx_chain_deposits_status (status),
    CONSTRAINT fk_chain_deposits_order FOREIGN KEY (order_id) REFERENCES orders (id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;
