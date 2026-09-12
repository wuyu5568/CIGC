-- CIGC 预售 P0 schema（勿用 GORM AutoMigrate，以本文件为准）
-- CREATE DATABASE cigc DEFAULT CHARSET utf8mb4;

CREATE TABLE IF NOT EXISTS users (
    id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    address            VARCHAR(64)     NOT NULL COMMENT 'wallet address lowercase',
    inviter_id         BIGINT UNSIGNED NULL COMMENT 'inviter user id, null for genesis',
    available_balance  DECIMAL(36, 8)  NOT NULL DEFAULT 0 COMMENT 'internal reward balance',
    frozen_balance     DECIMAL(36, 8)  NOT NULL DEFAULT 0 COMMENT 'withdraw freeze',
    paid_amount        DECIMAL(36, 8)  NOT NULL DEFAULT 0 COMMENT 'cumulative paid order amount',
    cap_effective      DECIMAL(36, 8)  NOT NULL DEFAULT 0 COMMENT 'daily match cap used by settle',
    disabled_at        DATETIME(3)     NULL COMMENT 'soft delete / disabled',
    created_at         DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at         DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_users_address (address),
    KEY idx_users_inviter (inviter_id),
    KEY idx_users_disabled (disabled_at),
    CONSTRAINT fk_users_inviter FOREIGN KEY (inviter_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

CREATE TABLE IF NOT EXISTS login_challenges (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    address     VARCHAR(64)     NOT NULL,
    nonce       VARCHAR(64)     NOT NULL,
    expires_at  DATETIME(3)     NOT NULL,
    used_at     DATETIME(3)     NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_login_challenges_nonce (nonce),
    KEY idx_login_challenges_address (address)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

CREATE TABLE IF NOT EXISTS user_recommends (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id    BIGINT UNSIGNED NOT NULL,
    path       VARCHAR(2048)   NOT NULL DEFAULT '' COMMENT 'ancestor ids e.g. 1,5,12',
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_user_recommends_user (user_id),
    KEY idx_user_recommends_path (path(255)),
    CONSTRAINT fk_user_recommends_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

CREATE TABLE IF NOT EXISTS packages (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    amount      DECIMAL(36, 8)  NOT NULL,
    title       VARCHAR(128)    NOT NULL,
    goods_desc  VARCHAR(512)    NOT NULL DEFAULT '',
    daily_cap   DECIMAL(36, 8)  NOT NULL COMMENT 'match cap when paid_amount reaches this package amount',
    sort_order  INT             NOT NULL DEFAULT 0,
    enabled     TINYINT(1)      NOT NULL DEFAULT 1,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_packages_amount (amount),
    KEY idx_packages_sort (sort_order, id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

INSERT INTO packages (amount, title, goods_desc, daily_cap, sort_order, enabled) VALUES
(1000,    '牙刷挖矿', '牙刷挖矿', 600,     10, 1),
(3000,    'AI眼镜挖矿', 'AI眼镜挖矿', 1800,    20, 1),
(6000,    '分布式存储芯片挖矿机', '分布式存储芯片挖矿机', 4000,    30, 1),
(12000,   '12,000 组合', '分布式存储芯片挖矿＋手机挖矿＋黄金钻石戒指＋多肽', 8000,    40, 1),
(24000,   '24,000 组合', '分布式存储芯片挖矿＋手机挖矿＋黄金钻石手链＋多肽', 16000,   50, 1),
(36000,   '36,000 组合', '分布式存储芯片挖矿＋手机挖矿＋黄金钻石项链＋多肽', 24000,   60, 1),
(50000,   '50,000 组合', '分布式存储芯片挖矿＋手机挖矿＋黄金钻石项链＋多肽', 30000,   70, 1),
(70000,   '70,000 组合', '分布式存储芯片挖矿＋手机挖矿＋黄金钻石项链＋多肽', 42000,   80, 1),
(100000,  '100,000 组合', '分布式存储芯片挖矿＋手机挖矿＋黄金钻石项链＋多肽', 60000,   90, 1),
(160000,  '160,000 组合', '分布式存储芯片挖矿＋手机挖矿＋黄金钻石项链＋多肽', 100000, 100, 1)
ON DUPLICATE KEY UPDATE title = VALUES(title), goods_desc = VALUES(goods_desc),
    daily_cap = VALUES(daily_cap), sort_order = VALUES(sort_order), enabled = VALUES(enabled);

CREATE TABLE IF NOT EXISTS orders (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id         BIGINT UNSIGNED NOT NULL,
    package_id      BIGINT UNSIGNED NOT NULL,
    amount          DECIMAL(36, 8)  NOT NULL,
    title_snapshot  VARCHAR(128)    NOT NULL,
    goods_snapshot  VARCHAR(512)    NOT NULL DEFAULT '',
    status          VARCHAR(16)     NOT NULL DEFAULT 'pending' COMMENT 'pending|confirming|paid|abnormal',
    tx_hash         VARCHAR(80)     NULL COMMENT 'set on chain match; NULL until paid',
    log_index       INT             NOT NULL DEFAULT 0,
    paid_at         DATETIME(3)     NULL,
    created_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_orders_user (user_id),
    KEY idx_orders_status (status),
    KEY idx_orders_package (package_id),
    UNIQUE KEY uk_orders_chain_event (tx_hash, log_index),
    CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_orders_package FOREIGN KEY (package_id) REFERENCES packages (id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

CREATE TABLE IF NOT EXISTS ledger_entries (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id      BIGINT UNSIGNED NOT NULL,
    order_id     BIGINT UNSIGNED NULL,
    entry_type   VARCHAR(32)     NOT NULL,
    amount       DECIMAL(36, 8)  NOT NULL,
    balance_kind VARCHAR(16)     NOT NULL COMMENT 'available | frozen',
    settle_date  DATE            NULL,
    remark       VARCHAR(255)    NOT NULL DEFAULT '',
    created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_ledger_user (user_id),
    KEY idx_ledger_order (order_id),
    KEY idx_ledger_type (entry_type),
    KEY idx_ledger_created (created_at),
    KEY idx_ledger_user_created (user_id, created_at),
    KEY idx_ledger_settle_date (settle_date),
    CONSTRAINT fk_ledger_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

CREATE TABLE IF NOT EXISTS withdraws (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id         BIGINT UNSIGNED NOT NULL,
    amount          DECIMAL(36, 8)  NOT NULL,
    fee_amount      DECIMAL(36, 8)  NOT NULL DEFAULT 0,
    credited_amount DECIMAL(36, 8)  NOT NULL DEFAULT 0,
    status          VARCHAR(16)     NOT NULL DEFAULT 'pending' COMMENT 'pending|rewarded|doing|pass|rejected|cancelled',
    remark          VARCHAR(255)    NOT NULL DEFAULT '',
    tx_hash         VARCHAR(80)     NOT NULL DEFAULT '',
    payout_error    VARCHAR(255)    NOT NULL DEFAULT '',
    reviewed_at     DATETIME(3)     NULL,
    created_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_withdraws_user (user_id),
    KEY idx_withdraws_status (status),
    CONSTRAINT fk_withdraws_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

CREATE TABLE IF NOT EXISTS business_configs (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    config_key VARCHAR(64)     NOT NULL,
    name       VARCHAR(128)    NOT NULL,
    value      TEXT            NOT NULL,
    sort_order INT             NOT NULL DEFAULT 0,
    updated_at TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_config_key (config_key),
    KEY idx_sort (sort_order, id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

INSERT INTO business_configs (config_key, name, value, sort_order) VALUES
('direct_rate', '直推奖励比例', '0.10', 10),
('match_rate', '对碰奖励比例', '0.10', 20),
('manage_rate', '三代管理奖励比例', '0.30', 30),
('min_withdraw_amount', '最低提现金额', '10', 40)
ON DUPLICATE KEY UPDATE name = VALUES(name), value = VALUES(value), sort_order = VALUES(sort_order);

-- 日结防重（金牛口径：上海自然日唯一占位）
CREATE TABLE IF NOT EXISTS settle_runs (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    settle_date    DATE            NOT NULL COMMENT 'Asia/Shanghai calendar day',
    forced         TINYINT(1)      NOT NULL DEFAULT 0,
    user_count     INT             NOT NULL DEFAULT 0,
    cap_updated    INT             NOT NULL DEFAULT 0,
    direct_count   INT             NOT NULL DEFAULT 0,
    match_count    INT             NOT NULL DEFAULT 0,
    manage_count   INT             NOT NULL DEFAULT 0,
    remark         VARCHAR(255)    NOT NULL DEFAULT '',
    created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_settle_runs_date (settle_date)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

-- 双轨安置（与推荐关系分离；注册按邀请顺序自动落位，管理端仍可手工指定）
CREATE TABLE IF NOT EXISTS user_placements (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id     BIGINT UNSIGNED NOT NULL COMMENT '被安置会员，一人一行',
    sponsor_id  BIGINT UNSIGNED NOT NULL COMMENT '安置上级（非推荐人）',
    side        CHAR(1)         NOT NULL COMMENT 'L|R 相对 sponsor 的直接区位',
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_placements_user (user_id),
    UNIQUE KEY uk_placements_sponsor_side (sponsor_id, side),
    KEY idx_placements_sponsor (sponsor_id),
    CONSTRAINT fk_placements_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_placements_sponsor FOREIGN KEY (sponsor_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

-- 对碰左右区结余（跨日；日结加当日新支付，配对后两边减 pair）
CREATE TABLE IF NOT EXISTS user_match_balances (
    user_id      BIGINT UNSIGNED NOT NULL PRIMARY KEY,
    left_remain  DECIMAL(36, 8)  NOT NULL DEFAULT 0,
    right_remain DECIMAL(36, 8)  NOT NULL DEFAULT 0,
    updated_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_match_bal_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

-- 一笔已支付订单的业绩只加进安置祖先一次（force 不重复累加）
CREATE TABLE IF NOT EXISTS match_order_applied (
    order_id    BIGINT UNSIGNED NOT NULL PRIMARY KEY,
    settle_date DATE            NOT NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_match_applied_order FOREIGN KEY (order_id) REFERENCES orders (id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;
