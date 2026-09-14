-- 套餐名称对齐金额档位（可重复执行）。订单名称跟当前套餐 title。
SET NAMES utf8mb4;
UPDATE packages SET title = '1,000 组合' WHERE amount = 1000;
UPDATE packages SET title = '3,000 组合' WHERE amount = 3000;
UPDATE packages SET title = '6,000 组合' WHERE amount = 6000;
UPDATE packages SET title = '12,000 组合' WHERE amount = 12000;
UPDATE packages SET title = '24,000 组合' WHERE amount = 24000;
UPDATE packages SET title = '36,000 组合' WHERE amount = 36000;
UPDATE packages SET title = '50,000 组合' WHERE amount = 50000;
UPDATE packages SET title = '70,000 组合' WHERE amount = 70000;
UPDATE packages SET title = '100,000 组合' WHERE amount = 100000;
UPDATE packages SET title = '160,000 组合' WHERE amount = 160000;

UPDATE orders o
INNER JOIN packages p ON p.id = o.package_id
SET o.title_snapshot = p.title
WHERE p.title <> '' AND o.title_snapshot <> p.title;
