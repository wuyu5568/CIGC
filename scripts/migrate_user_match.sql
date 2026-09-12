-- 已有库增量：对碰结余与订单业绩防重（新库由 schema.sql 创建）
CREATE TABLE IF NOT EXISTS user_match_balances (
    user_id      BIGINT UNSIGNED NOT NULL PRIMARY KEY,
    left_remain  DECIMAL(36, 8)  NOT NULL DEFAULT 0,
    right_remain DECIMAL(36, 8)  NOT NULL DEFAULT 0,
    updated_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_match_bal_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

CREATE TABLE IF NOT EXISTS match_order_applied (
    order_id    BIGINT UNSIGNED NOT NULL PRIMARY KEY,
    settle_date DATE            NOT NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_match_applied_order FOREIGN KEY (order_id) REFERENCES orders (id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;
