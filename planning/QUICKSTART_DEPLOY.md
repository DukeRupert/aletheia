# Quick Deployment Guide for angmar.dev

This is a streamlined guide for deploying Aletheia to your VPS at angmar.dev.

## Step 1: Prepare Local Environment

Create your production environment file:

```bash
# Copy example to production config
cp example.env .env.prod

# Edit with your production values
nano .env.prod
```

**Important values to set in `.env.prod`:**

```bash
# Docker Hub (must match what you pushed)
DOCKER_USERNAME=your-dockerhub-username
IMAGE_TAG=latest

# Database credentials
DB_USER=aletheia_user
DB_PASSWORD=<generate with: openssl rand -base64 32>
DB_NAME=aletheia_prod

# Security (CRITICAL!)
JWT_SECRET=<generate with: openssl rand -base64 64>

# Production settings
ENVIRONMENT=prod
LOG_LEVEL=info
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Storage (S3 recommended for production)
STORAGE_PROVIDER=s3
STORAGE_S3_BUCKET=your-bucket-name
STORAGE_S3_REGION=us-east-1
STORAGE_S3_BASE_URL=https://your-cloudfront-url.com
AWS_ACCESS_KEY_ID=your-key
AWS_SECRET_ACCESS_KEY=your-secret

# Email
EMAIL_PROVIDER=postmark
EMAIL_FROM_ADDRESS=noreply@angmar.dev
EMAIL_FROM_NAME=Aletheia
EMAIL_VERIFY_BASE_URL=https://aletheia.angmar.dev
POSTMARK_SERVER_TOKEN=your-token

# AI
ANTHROPIC_API_KEY=your-anthropic-key
```

## Step 2: Deploy Files to VPS

Use the deployment script:

```bash
# Make sure deploy.sh is executable
chmod +x deploy.sh

# Deploy files (or use: make deploy-files)
./deploy.sh
```

Or manually with rsync:

```bash
rsync -avz --progress \
    docker-compose.prod.yml \
    .env.prod \
    Caddyfile.angmar \
    dukerupert@angmar.dev:/home/dukerupert/aletheia/
```

## Step 3: Configure Caddy on VPS

SSH to your VPS:

```bash
ssh dukerupert@angmar.dev
cd /home/dukerupert/aletheia
```

Add Caddy configuration:

```bash
# Option 1: Add to main Caddyfile
sudo nano /etc/caddy/Caddyfile
# Copy contents from Caddyfile.angmar

# Option 2: Use sites-enabled pattern
sudo cp Caddyfile.angmar /etc/caddy/sites-enabled/aletheia
sudo nano /etc/caddy/Caddyfile
# Add: import sites-enabled/*

# Reload Caddy
sudo systemctl reload caddy
```

## Step 4: Start Services on VPS

Still on the VPS:

```bash
cd /home/dukerupert/aletheia

# Pull the latest image
docker compose -f docker-compose.prod.yml --env-file .env.prod pull

# Start services
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d

# Watch logs
docker compose -f docker-compose.prod.yml logs -f
```

## Step 5: Run Migrations

```bash
# Run database migrations
docker exec -it aletheia-app ./aletheiad -migrate-up
```

## Step 6: Verify Deployment

1. **Check services are running:**
   ```bash
   docker compose -f docker-compose.prod.yml ps
   ```

2. **Test the application:**
   ```bash
   # From VPS
   curl http://localhost:8080/health

   # From your local machine
   curl https://aletheia.angmar.dev
   ```

3. **Check logs:**
   ```bash
   docker compose -f docker-compose.prod.yml logs app
   ```

## Automatic Updates

Watchtower is configured to check for new images every 5 minutes. When you push a new image to Docker Hub:

1. Push to master branch (GitHub Actions builds and pushes automatically)
2. Wait ~5 minutes for Watchtower to detect and update
3. Check logs: `docker logs aletheia-watchtower`

## Useful Commands

```bash
# SSH to VPS and go to app directory
make deploy-ssh

# View logs
docker compose -f docker-compose.prod.yml logs -f app

# Restart services
docker compose -f docker-compose.prod.yml restart

# Stop services
docker compose -f docker-compose.prod.yml down

# Pull latest image manually
docker compose -f docker-compose.prod.yml pull

# Database backup
docker exec aletheia-postgres pg_dump -U aletheia_user aletheia_prod > backup_$(date +%Y%m%d).sql

# Access database
docker exec -it aletheia-postgres psql -U aletheia_user -d aletheia_prod
```

## Troubleshooting

**Container won't start:**
```bash
docker compose -f docker-compose.prod.yml logs app
```

**Port already in use:**
```bash
sudo netstat -tlnp | grep 8080
```

**Caddy issues:**
```bash
sudo systemctl status caddy
sudo journalctl -u caddy -f
```

**Reset everything:**
```bash
docker compose -f docker-compose.prod.yml down -v
docker compose -f docker-compose.prod.yml up -d
docker exec -it aletheia-app ./aletheiad -migrate-up
```

## Security Checklist

- [ ] Strong JWT_SECRET generated
- [ ] Strong database password set
- [ ] .env.prod has proper permissions (600)
- [ ] Firewall configured (ports 22, 80, 443 only)
- [ ] S3 bucket has proper permissions
- [ ] Email verification enabled
- [ ] HTTPS working via Caddy
- [ ] Regular backups configured

## Next Steps After Deployment

1. Create your first organization and user account
2. Test the complete workflow (upload photo, run AI analysis)
3. Configure monitoring/alerting
4. Set up automated database backups
5. Review logs for any errors
