#!/bin/sh

set -eu

if [ -f .env ]; then
    echo "Using existing .env file"
    exit 0
fi

umask 077

POSTGRES_PASSWORD=$(openssl rand -hex 16)
RABBITMQ_PASSWORD=$(openssl rand -hex 16)

cat > .env <<EOF
HTTP_ADDR=:8080
HTTP_PORT=8080

POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=tenant_app
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_DB=tenant_provisioning

RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_MANAGEMENT_PORT=15672
RABBITMQ_USER=tenant_app
RABBITMQ_PASSWORD=${RABBITMQ_PASSWORD}

WORKER_FAIL_RATE=0
WORKER_MIN_DELAY_MS=1000
WORKER_MAX_DELAY_MS=5000
EOF

echo "Local environment configuration created"