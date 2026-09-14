#!/usr/bin/env bash
# 在 AWS 主机 /opt/cigc 执行：装 Docker、开 swap、申请证书、拉起 compose。
set -euo pipefail

ROOT="${ROOT:-/opt/cigc}"
DOMAIN="${DOMAIN:-ispaygijdysxt.com}"
EMAIL="${CERT_EMAIL:-admin@${DOMAIN}}"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "run as root (sudo bash scripts/aws-bootstrap.sh)" >&2
  exit 1
fi

export DEBIAN_FRONTEND=noninteractive
apt-get update -y
apt-get install -y ca-certificates curl gnupg git rsync

if ! command -v docker >/dev/null 2>&1; then
  curl -fsSL https://get.docker.com | sh
fi
systemctl enable --now docker

if [[ ! -f /swapfile ]]; then
  fallocate -l 2G /swapfile
  chmod 600 /swapfile
  mkswap /swapfile
  swapon /swapfile
  echo '/swapfile none swap sw 0 0' >> /etc/fstab
fi

cd "$ROOT"
if [[ ! -f .env ]]; then
  echo "missing $ROOT/.env" >&2
  exit 1
fi

mkdir -p frontend/dapp frontend/admin
cp -f deploy/nginx/prod-http.conf deploy/nginx/active.conf

docker compose -f docker-compose.yml -f docker-compose.prod.yml --env-file .env up -d --build mysql api nginx

echo "waiting for API health"
for i in $(seq 1 40); do
  if docker compose exec -T api wget -qO- http://127.0.0.1:8000/health >/dev/null 2>&1; then
    break
  fi
  sleep 3
done

if [[ ! -d /var/lib/docker/volumes/cigc_certbot-etc/_data/live/$DOMAIN ]]; then
  docker compose -f docker-compose.yml -f docker-compose.prod.yml run --rm --no-deps \
    --entrypoint certbot \
    -v cigc_certbot-www:/var/www/certbot \
    -v cigc_certbot-etc:/etc/letsencrypt \
    nginx 2>/dev/null || true
  docker run --rm \
    -v cigc_certbot-www:/var/www/certbot \
    -v cigc_certbot-etc:/etc/letsencrypt \
    certbot/certbot certonly --webroot -w /var/www/certbot \
    -d "$DOMAIN" -d "www.$DOMAIN" \
    --agree-tos --email "$EMAIL" --non-interactive --keep-until-expiring || \
  docker run --rm \
    -v cigc_certbot-www:/var/www/certbot \
    -v cigc_certbot-etc:/etc/letsencrypt \
    certbot/certbot certonly --webroot -w /var/www/certbot \
    -d "$DOMAIN" \
    --agree-tos --email "$EMAIL" --non-interactive --keep-until-expiring
fi

if [[ -d /var/lib/docker/volumes/cigc_certbot-etc/_data/live/$DOMAIN ]]; then
  cp -f deploy/nginx/prod.conf deploy/nginx/active.conf
  docker compose -f docker-compose.yml -f docker-compose.prod.yml --env-file .env up -d nginx
fi

docker compose -f docker-compose.yml -f docker-compose.prod.yml --env-file .env ps
echo "ok  https://$DOMAIN/  https://$DOMAIN/admin/"
