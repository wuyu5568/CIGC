#!/usr/bin/env bash
# 冒烟：健康检查 + 公开接口 + 管理登录/配置。需要 API 已启动。
set -euo pipefail

API="${CIGC_SMOKE_API:-http://127.0.0.1:8000}"
NGINX="${CIGC_SMOKE_NGINX:-http://127.0.0.1}"
ADMIN_USER="${CIGC_ADMIN_USERNAME:-admin}"
ADMIN_PASS="${CIGC_ADMIN_PASSWORD:-admin123}"

fail() { echo "smoke fail: $*" >&2; exit 1; }

need() {
  local url="$1"
  local body
  body="$(curl -fsS "$url")" || fail "GET $url"
  echo "$body"
}

echo "smoke api=$API"

health="$(need "$API/health")"
echo "$health" | grep -q '"status":"ok"' || fail "api /health: $health"

pkg="$(need "$API/api/app_server/package_list")"
echo "$pkg" | grep -q '"status":"ok"' || fail "package_list: $pkg"
echo "$pkg" | grep -q 'release_tiers' || fail "package_list missing release_tiers"

price="$(need "$API/api/app_server/ispay_price")"
echo "$price" | grep -q '"price"' || fail "ispay_price: $price"

login="$(curl -fsS -H 'Content-Type: application/json' \
  -d "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\"}" \
  "$API/api/admin_cigc/login")" || fail "admin login"
echo "$login" | grep -q '"status":"ok"' || fail "admin login: $login"

token="$(printf '%s' "$login" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("token",""))' 2>/dev/null || true)"
if [ -z "$token" ]; then
  token="$(printf '%s' "$login" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')"
fi
[ -n "$token" ] || fail "no admin token"

cfg="$(curl -fsS -H "Authorization: Bearer $token" "$API/api/admin_cigc/config")" || fail "admin config"
echo "$cfg" | grep -q 'ispay_price' || fail "config missing ispay_price: $cfg"
echo "$cfg" | grep -q 'direct_rate' || fail "config missing direct_rate"

st="$(curl -fsS -H "Authorization: Bearer $token" "$API/api/admin_cigc/settle_status")" || fail "settle_status"
echo "$st" | grep -q '"settle_today"' || fail "settle_status: $st"

if curl -fsS "$NGINX/api/health" >/dev/null 2>&1; then
  echo "smoke nginx=$NGINX ok"
else
  echo "smoke nginx skipped (not listening)"
fi

echo "smoke ok"
