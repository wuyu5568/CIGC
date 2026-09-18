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
(cd "$ROOT" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o deploy/out/settle ./cmd/settle)

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
  --exclude .env --exclude .env.local --exclude '*.pem' --exclude '*.key' \
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
CIGC_ALLOW_FORCE_SETTLE=true
CIGC_FULL_DOWNLINE=false
CIGC_SETTLE_CRON=0 0 * * *
CIGC_SETTLE_TIMEZONE=Asia/Shanghai
CIGC_BSC_RPC=https://bsc-dataseed.binance.org/
CIGC_USDT_ADDRESS=0x55d398326f99059fF775485246999027B3197955
CIGC_ISPAY_ADDRESS=0xBF9b0594E110C381F2606961C78641a194999999
CIGC_BUY_CONTRACT=0x162bfFAcf7a89Bb6EbA05972C1DE0E1e97617c18
CIGC_DEPOSIT_CONFIRMATIONS=12
CIGC_DEPOSIT_CRON=* * * * *
CIGC_PAYOUT_ENABLED=false
DAPP_DIST=./frontend/dapp
ADMIN_DIST=./frontend/admin
ENV
fi
chmod 600 .env
python3 - <<'PY'
from pathlib import Path
p = Path('.env')
text = p.read_text()
lines = text.splitlines()
keys = {}
for line in lines:
    if '=' in line and not line.lstrip().startswith('#'):
        keys[line.split('=', 1)[0]] = True
out = list(lines)
if 'DAPP_DIST' not in keys:
    out.append('DAPP_DIST=./frontend/dapp')
if 'ADMIN_DIST' not in keys:
    out.append('ADMIN_DIST=./frontend/admin')
fixed = []
for line in out:
    if line.startswith('DAPP_DIST=') and 'placeholder' in line:
        fixed.append('DAPP_DIST=./frontend/dapp')
    elif line.startswith('ADMIN_DIST=') and 'placeholder' in line:
        fixed.append('ADMIN_DIST=./frontend/admin')
    else:
        fixed.append(line)
text = '\n'.join(fixed).rstrip() + '\n'
p.write_text(text)
p.chmod(0o600)
print('dist_env_ready')
PY
if [[ ! -f frontend/dapp/index.html || ! -f frontend/admin/index.html ]]; then
  echo "missing frontend dist at $REMOTE/frontend" >&2
  exit 1
fi
sudo bash scripts/aws-bootstrap.sh
EOF

echo "done"
echo "  site   https://ispaygijdysxt.com/"
echo "  admin  https://ispaygijdysxt.com/admin/"
echo "  admin user/pass 见服务器 $REMOTE/.env"
