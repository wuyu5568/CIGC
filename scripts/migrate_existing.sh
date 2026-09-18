#!/usr/bin/env bash
# 已有 MySQL 数据卷按顺序补增量。可重复执行。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HOST="${MYSQL_HOST:-127.0.0.1}"
PORT="${MYSQL_PORT:-3306}"
USER="${MYSQL_USER:-root}"
PASS="${MYSQL_PWD:-${MYSQL_ROOT_PASSWORD:-root}}"
DB="${MYSQL_DATABASE:-cigc}"

apply() {
  local path="$1"
  if command -v mysql >/dev/null 2>&1; then
    if mysql -h"$HOST" -P"$PORT" -u"$USER" -p"$PASS" --protocol=tcp "$DB" < "$path"; then
      return
    fi
  fi
  if command -v docker >/dev/null 2>&1; then
    docker compose -f "$ROOT/docker-compose.yml" exec -T mysql mysql -u"$USER" -p"$PASS" "$DB" < "$path"
    return
  fi
  echo "need mysql client on $HOST:$PORT or a running compose mysql" >&2
  exit 1
}

echo "migrate existing → $USER@$HOST:$PORT/$DB"
for f in \
  migrate_user_placements.sql \
  migrate_user_match.sql \
  migrate_chain_deposit.sql \
  migrate_ispay_static.sql \
  migrate_ispay_price.sql \
  migrate_lock_activate.sql \
  migrate_ispay_withdraw.sql \
  migrate_payout.sql \
  migrate_withdraw_fee.sql \
  migrate_withdraw_daily.sql \
  migrate_recharge.sql \
  migrate_package_days.sql \
  migrate_order_no.sql \
  migrate_package_titles.sql \
  migrate_package_image.sql \
  migrate_package_amount_days.sql \
  migrate_package_detail.sql \
  migrate_package_drop_amount_days.sql \
  migrate_package_contents.sql \
  migrate_package_skus.sql \
  migrate_package_sku_image.sql \
  migrate_shipping_addresses.sql \
  migrate_daily_cap_overflow.sql \
  migrate_ops_config.sql \
  migrate_daily_cap_tiers.sql \
  migrate_payout_max_ispay.sql
do
  echo "  apply $f"
  apply "$ROOT/scripts/$f"
done
echo "ok"
