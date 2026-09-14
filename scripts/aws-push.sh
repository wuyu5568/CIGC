#!/usr/bin/env bash
# 本机打包并 rsync 到 AWS。需要已能 ssh ubuntu@HOST。
set -euo pipefail

HOST="${AWS_HOST:-35.76.239.159}"
USER="${AWS_USER:-ubuntu}"
KEY="${AWS_KEY:-$HOME/.ssh/id_ed25519}"
REMOTE="${REMOTE_DIR:-/opt/cigc}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DAPP="$ROOT/dapp"
ADMIN="$ROOT/dapp-admin"

ssh_cmd=(ssh -i "$KEY" -o StrictHostKeyChecking=accept-new "$USER@$HOST")
rsync_ssh="ssh -i $KEY -o StrictHostKeyChecking=accept-new"

echo "build api linux/amd64"
mkdir -p "$ROOT/deploy/out"
(cd "$ROOT" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o deploy/out/app ./cmd/app)

echo "build dapp (mode cigc, same-origin /api)"
(cd "$DAPP" && npm ci --prefer-offline --no-audit --fund && npx vite build --mode cigc)
rm -rf "$ROOT/frontend/dapp"
mkdir -p "$ROOT/frontend/dapp"
cp -a "$DAPP/dist/." "$ROOT/frontend/dapp/"

echo "build admin (same-origin /api)"
(cd "$ADMIN" && npm ci --prefer-offline --no-audit --fund && \
  NODE_OPTIONS=--openssl-legacy-provider ADMIN_API= npx vue-cli-service build)
rm -rf "$ROOT/frontend/admin"
mkdir -p "$ROOT/frontend/admin"
cp -a "$ADMIN/dist/." "$ROOT/frontend/admin/"

echo "sync to $USER@$HOST:$REMOTE"
"${ssh_cmd[@]}" "sudo mkdir -p $REMOTE && sudo chown $USER:$USER $REMOTE"
rsync -az --delete \
  --exclude .git --exclude bin --exclude '*.log' \
  -e "$rsync_ssh" \
  "$ROOT/" "$USER@$HOST:$REMOTE/"

"${ssh_cmd[@]}" "bash -s" <<EOF
set -euo pipefail
cd $REMOTE
if [[ ! -f .env ]]; then
  PASS=\$(openssl rand -hex 16)
  JWT=\$(openssl rand -hex 24)
  cat > .env <<ENV
MYSQL_ROOT_PASSWORD=\$PASS
CIGC_JWT_KEY=\$JWT
CIGC_ADMIN_USERNAME=admin
CIGC_ADMIN_PASSWORD=admin123
CIGC_GENESIS_ADDRESS=0x8Bd86ad98D9fA366E52CFB5c08a3e33f5f412Fb1
CIGC_HTTP_ADDR=0.0.0.0:8000
CIGC_DATABASE_DSN=root:\$PASS@tcp(mysql:3306)/cigc?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai&time_zone=%27%2B08%3A00%27
CIGC_ALLOW_FORCE_SETTLE=false
CIGC_FULL_DOWNLINE=false
CIGC_SETTLE_CRON=0 0 * * *
CIGC_SETTLE_TIMEZONE=Asia/Shanghai
CIGC_BSC_RPC=https://bsc-dataseed.binance.org/
CIGC_USDT_ADDRESS=0x55d398326f99059fF775485246999027B3197955
CIGC_BUY_CONTRACT=0xCb63733FB936c7B3f147C757D383645e55769bF3
CIGC_DEPOSIT_CONFIRMATIONS=12
CIGC_PAYOUT_ENABLED=false
DAPP_DIST=./frontend/dapp
ADMIN_DIST=./frontend/admin
ENV
fi
chmod 600 .env
sudo bash scripts/aws-bootstrap.sh
EOF

echo "done"
echo "  site   https://ispaygijdysxt.com/"
echo "  admin  https://ispaygijdysxt.com/admin/"
echo "  admin user/pass 见服务器 $REMOTE/.env"
