#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
umask 077
mkdir -p generated
if [[ ! -f generated/app.env ]]; then
  db_password=$(openssl rand -hex 24)
  cat > generated/app.env <<ENV
MYSQL_DATABASE=autoops
MYSQL_ROOT_PASSWORD=$(openssl rand -hex 24)
MYSQL_USER=autoops
MYSQL_PASSWORD=$db_password
DB_HOST=mysql
DB_PORT=3306
DB_NAME=autoops
DB_USER=autoops
DB_PASSWORD=$db_password
REDIS_ADDR=redis:6379
REDIS_PASSWORD=$(openssl rand -hex 24)
AUTOOPS_INITIAL_ADMIN_PASSWORD=$(openssl rand -hex 12)
AUTOOPS_JWT_SECRET=$(openssl rand -hex 32)
WEB_ENCRYPTION_KEY=$(openssl rand -base64 32)
AIOPS_ENCRYPTION_KEY=$(openssl rand -base64 32)
AUTOOPS_SMTP_HOST=
AUTOOPS_SMTP_PORT=465
AUTOOPS_SMTP_USER=
AUTOOPS_SMTP_PASSWORD=
AUTOOPS_SMTP_FROM=
AUTOOPS_SMTP_TLS=tls
TZ=Asia/Shanghai
ENV
fi
[[ -f generated/config.yaml ]] || cp ../../docker/api/config.example.yaml generated/config.yaml
cp ../../api/sql/bootstrap.sql generated/bootstrap.sql
printf 'Prepared k8s/quickstart/generated (ignored by Git).\nReview app.env, then: kubectl apply -k k8s/quickstart\n'
