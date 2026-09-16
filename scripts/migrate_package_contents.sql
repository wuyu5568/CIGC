CREATE TABLE IF NOT EXISTS package_contents (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    package_id  BIGINT UNSIGNED NOT NULL,
    locale      VARCHAR(10)     NOT NULL,
    title       VARCHAR(128)    NOT NULL DEFAULT '',
    goods_desc  VARCHAR(512)    NOT NULL DEFAULT '',
    image       VARCHAR(512)    NOT NULL DEFAULT '',
    detail      MEDIUMTEXT,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_package_contents_locale (package_id, locale),
    KEY idx_package_contents_locale (locale),
    CONSTRAINT fk_package_contents_package FOREIGN KEY (package_id) REFERENCES packages (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET utf8mb4;

INSERT INTO package_contents (package_id, locale, title, goods_desc, image, detail)
SELECT id, 'zh', title, goods_desc, image, detail
FROM packages
ON DUPLICATE KEY UPDATE
    title = VALUES(title),
    goods_desc = VALUES(goods_desc),
    image = VALUES(image),
    detail = VALUES(detail);
