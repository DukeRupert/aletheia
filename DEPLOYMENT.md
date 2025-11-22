# Deployment Guide

This guide covers deploying Aletheia to a VPS using Docker, Docker Compose, and Watchtower for automatic updates.

## Overview

The deployment setup uses:
- **Docker** - Containerization
- **Docker Compose** - Multi-container orchestration
- **GitHub Actions** - CI/CD pipeline for testing and building
- **Docker Hub** - Container registry
- **Watchtower** - Automatic container updates
- **Caddy** - Reverse proxy with automatic HTTPS (running on VPS host)
- **PostgreSQL** - Database (containerized)

## Prerequisites

### On Your VPS

1. **Docker and Docker Compose installed**
   ```bash
   # Install Docker
   curl -fsSL https://get.docker.com -o get-docker.sh
   sudo sh get-docker.sh

   # Add your user to docker group
   sudo usermod -aG docker $USER

   # Install Docker Compose
   sudo apt-get update
   sudo apt-get install docker-compose-plugin
   ```

2. **Caddy installed and running**
   ```bash
   # Install Caddy
   sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
   curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
   curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
   sudo apt update
   sudo apt install caddy
   ```

3. **Domain name** pointed to your VPS IP address

### GitHub Secrets

Configure these secrets in your GitHub repository (Settings → Secrets and variables → Actions):

- `DOCKER_USERNAME` - Your Docker Hub username
- `DOCKER_PASSWORD` - Your Docker Hub password or access token

## Initial Setup

### 1. Clone Repository on VPS

```bash
mkdir -p ~/apps
cd ~/apps
git clone https://github.com/yourusername/aletheia.git
cd aletheia
```

### 2. Create Production Environment File

Create a `.env.prod` file with your production configuration:

```bash
# Copy the example and edit
cat > .env.prod <<EOF
# Docker configuration
DOCKER_USERNAME=your-dockerhub-username
IMAGE_TAG=latest

# Database credentials
DB_USER=aletheia_user
DB_PASSWORD=CHANGE_THIS_SECURE_PASSWORD
DB_NAME=aletheia_prod

# Security (REQUIRED - generate a strong secret)
JWT_SECRET=CHANGE_THIS_TO_A_LONG_RANDOM_STRING

# Storage - S3 Configuration (recommended for production)
STORAGE_PROVIDER=s3
STORAGE_S3_BUCKET=your-s3-bucket-name
STORAGE_S3_REGION=us-east-1
STORAGE_S3_BASE_URL=https://your-cloudfront-domain.com
AWS_ACCESS_KEY_ID=your-aws-access-key
AWS_SECRET_ACCESS_KEY=your-aws-secret-key

# Email - Postmark Configuration
EMAIL_PROVIDER=postmark
EMAIL_FROM_ADDRESS=noreply@yourdomain.com
EMAIL_FROM_NAME=Aletheia
EMAIL_VERIFY_BASE_URL=https://yourdomain.com
POSTMARK_SERVER_TOKEN=your-postmark-token

# Queue Configuration
QUEUE_WORKER_COUNT=3
QUEUE_POLL_INTERVAL=1s
QUEUE_JOB_TIMEOUT=60s
QUEUE_ENABLE_RATE_LIMITING=true

# AI Configuration
ANTHROPIC_API_KEY=your-anthropic-api-key

# Watchtower Configuration (check for updates every 5 minutes)
WATCHTOWER_POLL_INTERVAL=300
EOF
```

**Important:** Edit this file and replace all placeholder values with your actual credentials.

### 3. Generate Strong Secrets

```bash
# Generate JWT secret
openssl rand -base64 64

# Generate database password
openssl rand -base64 32
```

### 4. Configure Caddy

Add your domain configuration to Caddy. See `Caddyfile.example` for reference.

```bash
# Edit your Caddy configuration
sudo nano /etc/caddy/Caddyfile
```

Add:
```caddy
yourdomain.com {
    reverse_proxy localhost:1323
    encode gzip

    log {
        output file /var/log/caddy/aletheia-access.log
        format json
    }

    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
        X-Frame-Options "SAMEORIGIN"
        X-XSS-Protection "1; mode=block"
        X-Content-Type-Options "nosniff"
        Referrer-Policy "strict-origin-when-cross-origin"
        -Server
    }
}
```

```bash
# Reload Caddy
sudo systemctl reload caddy
```

### 5. Start the Application

```bash
# Load environment variables and start services
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d

# Check logs
docker compose -f docker-compose.prod.yml logs -f
```

### 6. Run Database Migrations

```bash
# Run migrations inside the container
docker exec -it aletheia-app ./aletheia -migrate-up

# Or connect to the database directly
docker exec -it aletheia-postgres psql -U aletheia_user -d aletheia_prod
```

### 7. Seed Initial Data (Optional)

If you have seed scripts or need to create initial safety codes:

```bash
# Execute seed commands
docker exec -it aletheia-app ./aletheia -seed
```

## CI/CD Pipeline

The GitHub Actions workflow (`.github/workflows/ci-cd.yml`) automatically:

1. **On Pull Requests**: Runs tests
2. **On Push to Master**: Runs tests + builds and pushes Docker image to Docker Hub
3. **On Version Tags** (`v*`): Builds and tags with semantic version

### Workflow Stages

1. **Test Job**
   - Sets up Go environment
   - Starts PostgreSQL test database
   - Runs `go vet`, `go fmt`, and unit tests
   - Uploads coverage to Codecov (optional)

2. **Build Job** (only on master branch or tags)
   - Builds Docker image
   - Tags with branch name, git SHA, or semantic version
   - Pushes to Docker Hub
   - Uses layer caching for faster builds

### Triggering a Deployment

```bash
# Option 1: Push to master (automatic deployment via Watchtower)
git push origin master

# Option 2: Create a version tag
git tag v1.0.0
git push origin v1.0.0
```

Watchtower will automatically:
- Detect the new image on Docker Hub (checks every 5 minutes by default)
- Pull the latest image
- Recreate the container with zero downtime
- Clean up old images

## Monitoring and Maintenance

### View Logs

```bash
# All services
docker compose -f docker-compose.prod.yml logs -f

# Specific service
docker compose -f docker-compose.prod.yml logs -f app

# Last 100 lines
docker compose -f docker-compose.prod.yml logs --tail=100 app
```

### Check Service Status

```bash
docker compose -f docker-compose.prod.yml ps
```

### Manual Container Updates

```bash
# Pull latest image
docker compose -f docker-compose.prod.yml --env-file .env.prod pull

# Recreate containers
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d

# Remove old images
docker image prune -f
```

### Database Backup

```bash
# Create backup
docker exec aletheia-postgres pg_dump -U aletheia_user aletheia_prod > backup_$(date +%Y%m%d).sql

# Restore from backup
docker exec -i aletheia-postgres psql -U aletheia_user -d aletheia_prod < backup_20250101.sql
```

### Access Database

```bash
# PostgreSQL shell
docker exec -it aletheia-postgres psql -U aletheia_user -d aletheia_prod

# Run SQL file
docker exec -i aletheia-postgres psql -U aletheia_user -d aletheia_prod < script.sql
```

## Updating Configuration

1. Edit `.env.prod` file
2. Recreate containers:
   ```bash
   docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
   ```

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker compose -f docker-compose.prod.yml logs app

# Check if port is already in use
sudo netstat -tlnp | grep 1323

# Restart services
docker compose -f docker-compose.prod.yml restart
```

### Database Connection Issues

```bash
# Check if PostgreSQL is running
docker compose -f docker-compose.prod.yml ps postgres

# Test database connection
docker exec -it aletheia-postgres psql -U aletheia_user -d aletheia_prod -c "SELECT 1;"

# Check database logs
docker compose -f docker-compose.prod.yml logs postgres
```

### Watchtower Not Updating

```bash
# Check Watchtower logs
docker logs aletheia-watchtower

# Manually trigger update
docker exec aletheia-watchtower /watchtower --run-once

# Verify image labels
docker inspect aletheia-app | grep watchtower
```

### Health Check Failing

```bash
# Check health status
docker inspect aletheia-app | grep -A 10 Health

# Test health endpoint manually
curl http://localhost:1323/health
```

## Security Best Practices

1. **Use strong, unique passwords** for all services
2. **Keep secrets in `.env.prod`** and never commit to git
3. **Regularly update containers** (Watchtower handles this)
4. **Enable firewall** and only expose necessary ports:
   ```bash
   sudo ufw allow 22/tcp    # SSH
   sudo ufw allow 80/tcp    # HTTP
   sudo ufw allow 443/tcp   # HTTPS
   sudo ufw enable
   ```
5. **Regular database backups** (automate with cron)
6. **Monitor logs** for suspicious activity
7. **Use S3 for production storage** instead of local filesystem
8. **Enable rate limiting** in your application and Caddy

## Performance Optimization

1. **Adjust worker count** based on VPS resources:
   ```bash
   # In .env.prod
   QUEUE_WORKER_COUNT=5  # Increase for more concurrent jobs
   ```

2. **Database connection pooling** is configured in the app (25 max connections)

3. **Enable Caddy caching** for static assets (add to Caddyfile if needed)

4. **Monitor resource usage**:
   ```bash
   docker stats
   ```

## Rolling Back

If a deployment causes issues:

```bash
# Option 1: Use a specific image tag
# Edit .env.prod and set IMAGE_TAG to a previous version
IMAGE_TAG=v1.0.0

# Recreate container
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d

# Option 2: Pause Watchtower and manually pull previous image
docker pause aletheia-watchtower
docker pull your-dockerhub-username/aletheia:v1.0.0
docker tag your-dockerhub-username/aletheia:v1.0.0 your-dockerhub-username/aletheia:latest
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
docker unpause aletheia-watchtower
```

## Additional Resources

- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Caddy Documentation](https://caddyserver.com/docs/)
- [Watchtower Documentation](https://containrrr.dev/watchtower/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)

## Support

For issues or questions:
- Check application logs: `docker compose -f docker-compose.prod.yml logs -f app`
- Review this deployment guide
- Check GitHub Issues for known problems
