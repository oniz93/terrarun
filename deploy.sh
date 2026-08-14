#!/usr/bin/env bash
set -euo pipefail

export COMPOSE_PROJECT_NAME=terrarun

if [ ! -f .env ]; then
    echo "ERROR: .env file not found. Copy .env.example to .env and fill in secrets."
    exit 1
fi

source .env

echo "=== Pulling latest code ==="
git pull origin main

echo "=== Building and starting services ==="
docker compose -f docker-compose.prod.yml build
docker compose -f docker-compose.prod.yml up -d postgres redis rustfs

echo "=== Waiting for dependencies ==="
sleep 10

echo "=== Running database migrations ==="
docker compose -f docker-compose.prod.yml run --rm migrate

echo "=== Starting API ==="
docker compose -f docker-compose.prod.yml up -d api

echo "=== Starting nginx ==="
docker compose -f docker-compose.prod.yml up -d nginx

echo "=== Initializing Let's Encrypt (first run only) ==="
DOMAIN="${DOMAIN:-terrarun.app}"
EMAIL="${EMAIL:-admin@${DOMAIN}}"
if [ ! -d "certbot_conf/live/$DOMAIN" ]; then
    docker compose -f docker-compose.prod.yml run --rm certbot certonly \
        --webroot -w /var/www/certbot \
        -d "$DOMAIN" \
        --email "$EMAIL" \
        --agree-tos \
        --non-interactive

    echo "Restarting nginx to load new certificates..."
    docker compose -f docker-compose.prod.yml exec nginx nginx -s reload
fi

echo "=== Starting certbot renewal daemon ==="
docker compose -f docker-compose.prod.yml up -d certbot

echo ""
echo "=== Deployment complete ==="
echo "API: https://$DOMAIN"
echo ""
echo "Monitor with: docker compose -f docker-compose.prod.yml logs -f api"
