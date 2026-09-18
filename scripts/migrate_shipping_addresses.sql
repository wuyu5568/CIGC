CREATE TABLE IF NOT EXISTS shipping_addresses (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id     BIGINT UNSIGNED NOT NULL COMMENT 'one address per user',
    name        VARCHAR(64)     NOT NULL DEFAULT '',
    contact     VARCHAR(64)     NOT NULL DEFAULT '' COMMENT 'phone or other contact',
    address     VARCHAR(512)    NOT NULL DEFAULT '',
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_shipping_addresses_user (user_id),
    CONSTRAINT fk_shipping_addresses_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;
