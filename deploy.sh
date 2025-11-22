#!/bin/bash
# Deploy Aletheia to VPS
# Usage: ./deploy.sh

set -e

VPS_HOST="angmar.dev"
VPS_USER="dukerupert"
VPS_PATH="/home/dukerupert/aletheia"

echo "🚀 Deploying Aletheia to $VPS_HOST..."

# Check if .env.prod exists
if [ ! -f .env.prod ]; then
    echo "❌ Error: .env.prod file not found!"
    echo "📝 Please create .env.prod from example.env first:"
    echo "   cp example.env .env.prod"
    echo "   # Then edit .env.prod with your production values"
    exit 1
fi

# Create directory on VPS if it doesn't exist
echo "📁 Creating deployment directory on VPS..."
ssh ${VPS_USER}@${VPS_HOST} "mkdir -p ${VPS_PATH}"

# Copy deployment files
echo "📤 Copying deployment files..."
rsync -avz --progress \
    docker-compose.prod.yml \
    .env.prod \
    Caddyfile.example \
    Caddyfile.angmar \
    ${VPS_USER}@${VPS_HOST}:${VPS_PATH}/

echo "✅ Files copied successfully!"
echo ""
echo "📋 Next steps on your VPS:"
echo "   1. SSH to your VPS: ssh ${VPS_USER}@${VPS_HOST}"
echo "   2. cd ${VPS_PATH}"
echo "   3. Review and edit .env.prod with production values"
echo "   4. Configure Caddy (see Caddyfile.example)"
echo "   5. Start services: docker compose -f docker-compose.prod.yml --env-file .env.prod up -d"
echo "   6. Check logs: docker compose -f docker-compose.prod.yml logs -f"
echo ""
echo "🎉 Deployment preparation complete!"
