#!/usr/bin/env bash
# 在 /opt/cigc 上执行：清测试数据，只留创世地址。
set -euo pipefail

ROOT="${ROOT:-/opt/cigc}"
cd "$ROOT"
if [[ ! -f .env ]]; then
  echo "missing $ROOT/.env" >&2
  exit 1
fi

GEN="$(grep -E '^CIGC_GENESIS_ADDRESS=' .env | head -1 | cut -d= -f2- | tr -d '\"' | tr 'A-F' 'a-f' | tr -d '[:space:]')"
if [[ ! "$GEN" =~ ^0x[0-9a-f]{40}$ ]]; then
  echo "invalid CIGC_GENESIS_ADDRESS" >&2
  exit 1
fi

SQL="$(mktemp)"
trap 'rm -f "$SQL"' EXIT
{
  printf "SET @genesis_address := '%s';\n" "$GEN"
  cat "$ROOT/scripts/reset_test_data.sql"
} > "$SQL"

docker compose -f docker-compose.yml -f docker-compose.prod.yml --env-file .env exec -T mysql \
  sh -c 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" cigc' < "$SQL"
