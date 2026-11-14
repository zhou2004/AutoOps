#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
if [[ ! -f .env ]]; then
  umask 077
  cat > .env <<ENV
AUTOOPS_IMAGE_TAG=v1.1.0
AUTOOPS_IMAGE_REGISTRY=feelzhou
MYSQL_DATABASE=autoops
MYSQL_ROOT_PASSWORD=$(openssl rand -hex 24)
MYSQL_PASSWORD=$(openssl rand -hex 24)
REDIS_PASSWORD=$(openssl rand -hex 24)
AUTOOPS_INITIAL_ADMIN_PASSWORD=$(openssl rand -hex 12)
AUTOOPS_JWT_SECRET=$(openssl rand -hex 32)
WEB_ENCRYPTION_KEY=$(openssl rand -base64 32)
AIOPS_ENCRYPTION_KEY=$(openssl rand -base64 32)
WEB_PORT=8080
TZ=Asia/Shanghai
AUTOOPS_SMTP_HOST=
AUTOOPS_SMTP_PORT=465
AUTOOPS_SMTP_USER=
AUTOOPS_SMTP_PASSWORD=
AUTOOPS_SMTP_FROM=
AUTOOPS_SMTP_TLS=tls
ENV
  echo 'Created docker/.env with unique local passwords. Read AUTOOPS_INITIAL_ADMIN_PASSWORD for the first login.'
else
  echo 'Keeping existing docker/.env. See .env.example for required settings.'
fi
if [[ ! -f api/config.yaml ]]; then cp api/config.example.yaml api/config.yaml; chmod 600 api/config.yaml; fi
printf 'Next: docker compose config --quiet && docker compose up -d\n'
