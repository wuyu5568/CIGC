-- 已有库增量：安置表（新库由 schema.sql 创建；注册按邀请顺序自动落位）
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
