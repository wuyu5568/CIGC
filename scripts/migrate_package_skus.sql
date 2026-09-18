CREATE TABLE IF NOT EXISTS package_skus (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    package_id  BIGINT UNSIGNED NOT NULL,
    name        VARCHAR(128)    NOT NULL DEFAULT '',
    name_en     VARCHAR(128)    NOT NULL DEFAULT '',
    amount      DECIMAL(36, 8)  NOT NULL DEFAULT 0,
    sort_order  INT             NOT NULL DEFAULT 0,
    enabled     TINYINT(1)      NOT NULL DEFAULT 1,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_package_skus_package (package_id, sort_order, id),
    CONSTRAINT fk_package_skus_package FOREIGN KEY (package_id) REFERENCES packages (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;
