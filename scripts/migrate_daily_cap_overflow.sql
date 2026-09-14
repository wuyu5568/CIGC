-- 当日动态奖累计封顶：已用额度 + 超额冻结（次日 0:00 封账，再过 overflow_clear_hours 小时清除，默认 72）
CREATE TABLE IF NOT EXISTS user_daily_dynamic (
    user_id     BIGINT UNSIGNED NOT NULL,
    settle_date DATE            NOT NULL,
    used        DECIMAL(36, 8)  NOT NULL DEFAULT 0 COMMENT 'today dynamic USDT-value under cap',
    PRIMARY KEY (user_id, settle_date),
    CONSTRAINT fk_daily_dynamic_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

CREATE TABLE IF NOT EXISTS cap_overflow_holds (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id      BIGINT UNSIGNED NOT NULL,
    value        DECIMAL(36, 8)  NOT NULL COMMENT 'USDT-value before U/ispay split',
    usdt         DECIMAL(36, 8)  NOT NULL,
    ispay        DECIMAL(36, 8)  NOT NULL,
    source_type  VARCHAR(32)     NOT NULL COMMENT 'direct|match|manage|static|inactive|admin',
    order_id     BIGINT UNSIGNED NULL,
    settle_date  DATE            NOT NULL,
    remark       VARCHAR(255)    NOT NULL DEFAULT '',
    created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    expires_at   DATETIME(3)     NULL COMMENT 'set at next 00:00 packaging; settle_date + 4 days',
    released_at  DATETIME(3)     NULL COMMENT 'unlocked by buying an order (package daily_cap FIFO)',
    burned_at    DATETIME(3)     NULL COMMENT 'cleared at packaged expiry, no available credit',
    KEY idx_overflow_user (user_id),
    KEY idx_overflow_expire (expires_at, burned_at, released_at),
    CONSTRAINT fk_overflow_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

-- 已有库：未封账批次 expires_at 为空；旧 24h TTL 改回未封账，等下次 0:00 按 settle_date+4 重写
ALTER TABLE cap_overflow_holds
    MODIFY expires_at DATETIME(3) NULL COMMENT 'set at next 00:00 packaging; settle_date + 4 days',
    MODIFY released_at DATETIME(3) NULL COMMENT 'unlocked by buying an order (package daily_cap FIFO)',
    MODIFY burned_at DATETIME(3) NULL COMMENT 'cleared at packaged expiry, no available credit';

UPDATE cap_overflow_holds
SET expires_at = NULL
WHERE released_at IS NULL
  AND burned_at IS NULL
  AND value > 0
  AND expires_at IS NOT NULL
  AND expires_at < DATE_ADD(settle_date, INTERVAL 4 DAY);
