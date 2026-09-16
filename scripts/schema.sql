-- CIGC 预售 P0 schema（勿用 GORM AutoMigrate，以本文件为准）
-- CREATE DATABASE cigc DEFAULT CHARSET utf8mb4;

CREATE TABLE IF NOT EXISTS users (
    id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    address            VARCHAR(64)     NOT NULL COMMENT 'wallet address lowercase',
    inviter_id         BIGINT UNSIGNED NULL COMMENT 'inviter user id, null for genesis',
    available_balance  DECIMAL(36, 8)  NOT NULL DEFAULT 0 COMMENT 'internal reward balance',
    recharge_balance   DECIMAL(36, 8)  NOT NULL DEFAULT 0 COMMENT 'buy-only USDT from recharge page',
    frozen_balance     DECIMAL(36, 8)  NOT NULL DEFAULT 0 COMMENT 'usdt withdraw freeze',
    frozen_ispay       DECIMAL(36, 8)  NOT NULL DEFAULT 0 COMMENT 'ispay withdraw freeze',
    ispay_balance      DECIMAL(36, 8)  NOT NULL DEFAULT 0 COMMENT 'ispay coin balance',
    lock_balance       DECIMAL(36, 8)  NOT NULL DEFAULT 0 COMMENT 'inactive reward + daily-cap overflow USDT',
    lock_ispay         DECIMAL(36, 8)  NOT NULL DEFAULT 0 COMMENT 'inactive reward + daily-cap overflow ispay',
    paid_amount        DECIMAL(36, 8)  NOT NULL DEFAULT 0 COMMENT 'cumulative paid order amount',
    cap_effective      DECIMAL(36, 8)  NOT NULL DEFAULT 0 COMMENT 'daily dynamic cap from max paid order',
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
    daily_cap     DECIMAL(36, 8)  NOT NULL DEFAULT 0 COMMENT 'legacy column; match cap is computed from order amount',
    release_days  INT             NOT NULL DEFAULT 300 COMMENT '300|600|750 default static release',
    sort_order    INT             NOT NULL DEFAULT 0,
    enabled     TINYINT(1)      NOT NULL DEFAULT 1,
    image       VARCHAR(512)    NOT NULL DEFAULT '',
    detail      MEDIUMTEXT      NULL COMMENT 'web3 goods rich text detail',
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_packages_sort (sort_order, id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

INSERT INTO packages (amount, title, goods_desc, daily_cap, sort_order, enabled)
SELECT v.amount, v.title, v.goods_desc, v.daily_cap, v.sort_order, v.enabled
FROM (
    SELECT 1000 AS amount, CAST('1,000 组合' AS CHAR(128)) AS title, CAST('牙刷挖矿' AS CHAR(512)) AS goods_desc, 600 AS daily_cap, 10 AS sort_order, 1 AS enabled
    UNION ALL SELECT 3000, '3,000 组合', 'AI眼镜挖矿', 1800, 20, 1
    UNION ALL SELECT 6000, '6,000 组合', '分布式存储芯片挖矿机', 4000, 30, 1
    UNION ALL SELECT 12000, '12,000 组合', '分布式存储芯片挖矿＋手机挖矿＋黄金钻石戒指＋多肽', 8000, 40, 1
    UNION ALL SELECT 24000, '24,000 组合', '分布式存储芯片挖矿＋手机挖矿＋黄金钻石手链＋多肽', 16000, 50, 1
    UNION ALL SELECT 36000, '36,000 组合', '分布式存储芯片挖矿＋手机挖矿＋黄金钻石项链＋多肽', 24000, 60, 1
    UNION ALL SELECT 50000, '50,000 组合', '分布式存储芯片挖矿＋手机挖矿＋黄金钻石项链＋多肽', 30000, 70, 1
    UNION ALL SELECT 70000, '70,000 组合', '分布式存储芯片挖矿＋手机挖矿＋黄金钻石项链＋多肽', 42000, 80, 1
    UNION ALL SELECT 100000, '100,000 组合', '分布式存储芯片挖矿＋手机挖矿＋黄金钻石项链＋多肽', 60000, 90, 1
    UNION ALL SELECT 160000, '160,000 组合', '分布式存储芯片挖矿＋手机挖矿＋黄金钻石项链＋多肽', 100000, 100, 1
) v
WHERE NOT EXISTS (SELECT 1 FROM packages LIMIT 1);

CREATE TABLE IF NOT EXISTS orders (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    order_no        VARCHAR(32)     NULL,
    user_id         BIGINT UNSIGNED NOT NULL,
    package_id      BIGINT UNSIGNED NOT NULL,
    amount          DECIMAL(36, 8)  NOT NULL,
    title_snapshot  VARCHAR(128)    NOT NULL,
    goods_snapshot  VARCHAR(512)    NOT NULL DEFAULT '',
    status          VARCHAR(16)     NOT NULL DEFAULT 'pending' COMMENT 'pending|confirming|paid|abnormal',
    tx_hash         VARCHAR(80)     NULL COMMENT 'set on chain match; NULL until paid',
    log_index       INT             NOT NULL DEFAULT 0,
    release_days    INT             NOT NULL DEFAULT 0 COMMENT '300|600|750 static release',
    paid_at         DATETIME(3)     NULL,
    created_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_orders_user (user_id),
    KEY idx_orders_status (status),
    KEY idx_orders_package (package_id),
    UNIQUE KEY uk_orders_no (order_no),
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
    asset           VARCHAR(16)     NOT NULL DEFAULT 'usdt' COMMENT 'usdt|ispay',
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
('direct_rate', '直推奖比例', '0.10', 10),
('match_rate', '对碰奖比例', '0.10', 20),
('manage_rate', '管理奖比例', '0.30', 30),
('manage_generations', '管理奖代数', '3', 31),
('min_withdraw_amount', 'USDT 单笔最低提现', '10', 40),
('min_withdraw_amount_ispay', 'ISPAY 单笔最低提现', '0', 41),
('withdraw_fee_rate', 'USDT 提现手续费', '0.10', 42),
('withdraw_fee_rate_ispay', 'ISPAY 提现手续费', '0', 43),
('withdraw_daily_limit', 'USDT 单笔提现上限', '1000', 44),
('withdraw_daily_limit_ispay', 'ISPAY 单笔提现上限', '1000', 45),
('ispay_price', 'ISPAY 测试现价', '2000', 50),
('overflow_clear_hours', '冻结清除时间', '72', 60),
('withdraw_enabled', '提现开关', '1', 70),
('daily_cap_tiers', '日封顶档位', '[{"max_amount":"3000","daily_cap":"600"},{"max_amount":"6000","daily_cap":"1800"},{"max_amount":"12000","daily_cap":"4000"},{"max_amount":"24000","daily_cap":"8000"},{"max_amount":"36000","daily_cap":"16000"},{"max_amount":"50000","daily_cap":"24000"},{"max_amount":"70000","daily_cap":"30000"},{"max_amount":"100000","daily_cap":"42000"},{"max_amount":"160000","daily_cap":"60000"},{"max_amount":"","daily_cap":"100000"}]', 80)
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
    static_count   INT             NOT NULL DEFAULT 0,
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

-- 当日动态奖已计入封顶的产值（上海自然日）
CREATE TABLE IF NOT EXISTS user_daily_dynamic (
    user_id     BIGINT UNSIGNED NOT NULL,
    settle_date DATE            NOT NULL,
    used        DECIMAL(36, 8)  NOT NULL DEFAULT 0 COMMENT 'today dynamic USDT-value under cap',
    PRIMARY KEY (user_id, settle_date),
    CONSTRAINT fk_daily_dynamic_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

-- 超过当日动态封顶的冻结；次日 0:00 封账，再过 overflow_clear_hours 小时清除（默认 72）
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

-- 链上扫块游标
CREATE TABLE IF NOT EXISTS chain_scan_cursors (
    name         VARCHAR(64)     NOT NULL PRIMARY KEY,
    block_number BIGINT UNSIGNED NOT NULL DEFAULT 0,
    updated_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

-- 链上入账事件（匹配成功写 order；对不上为 abnormal，不入账）
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
