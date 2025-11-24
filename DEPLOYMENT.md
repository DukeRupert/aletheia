# Deployment Guide - Simple MVP Setup

Minimal CI/CD pipeline for Aletheia MVP.

## Overview

**GitHub Actions** → **Docker Hub** → **Manual Deploy on Server**

- GitHub Actions runs tests and builds Docker images automatically
- Images pushed to Docker Hub
- You manually deploy when ready (keeps it simple for MVP)

## One-Time Setup

### 1. GitHub Secrets (Required)

Go to GitHub repo: **Settings → Secrets and variables → Actions**

Add these secrets:
- `DOCKER_USERNAME` - Your Docker Hub username
- `DOCKER_PASSWORD` - Docker Hub access token (create at hub.docker.com/settings/security)

### 2. Server Prerequisites

Install Docker on your server:
```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Install Docker Compose plugin
sudo apt-get update
sudo apt-get install docker-compose-plugin
```

## How It Works

### Automatic Builds (GitHub Actions)

The workflow at `.github/workflows/ci-cd.yml` automatically:

**On every push/PR:**
- Runs tests with PostgreSQL
- Runs `go vet` and `go fmt` checks

**On push to `master`:**
- Builds Docker image
- Pushes to Docker Hub as `yourusername/aletheia:master`

**On version tags** (e.g., `v1.0.0`):
- Builds Docker image
- Pushes to Docker Hub with multiple tags:
  - `yourusername/aletheia:1.0.0`
  - `yourusername/aletheia:1.0`
  - `yourusername/aletheia:1`

### Deploy to Server

**First time setup on server:**

1. Create `docker-compose.prod.yml`:

```yaml
services:
  app:
    image: yourusername/aletheia:latest  # Replace with your Docker Hub username
    container_name: aletheia-app
    ports:
      - "1323:1323"
    env_file:
      - .env
    environment:
      DB_HOSTNAME: postgres
      STORAGE_LOCAL_PATH: /app/uploads
    volumes:
      - uploads:/app/uploads
    networks:
      - app-network
    restart: always
    depends_on:
      - postgres

  postgres:
    image: postgres:16-alpine
    container_name: aletheia-db
    environment:
      POSTGRES_DB: aletheia
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - app-network
    restart: always

volumes:
  uploads:
  postgres_data:

networks:
  app-network:
```

2. Create `.env` file with production config:

```bash
# Server
ENVIRONMENT=prod
SERVER_PORT=1323

# Database
DB_USER=postgres
DB_PASSWORD=your-secure-password-here
DB_HOSTNAME=postgres
DB_PORT=5432
DB_NAME=aletheia

# Security (REQUIRED)
JWT_SECRET=your-secure-jwt-secret-here

# Storage (local for MVP, can switch to S3 later)
STORAGE_PROVIDER=local
STORAGE_LOCAL_PATH=/app/uploads
STORAGE_LOCAL_URL=https://yourdomain.com/uploads

# Queue
QUEUE_PROVIDER=postgres
QUEUE_WORKER_COUNT=3

# Add other vars as needed (email, AI, etc.)
```

3. Generate secrets:

```bash
# JWT secret
openssl rand -base64 64

# DB password
openssl rand -base64 32
```

## Deploying Updates

### Step 1: Push code to GitHub

```bash
# For latest changes
git push origin master

# OR create a version tag
git tag v1.0.0
git push origin v1.0.0
```

GitHub Actions will automatically build and push to Docker Hub.

### Step 2: Deploy on server

```bash
# Pull latest image
docker compose -f docker-compose.prod.yml pull

# Restart with new image
docker compose -f docker-compose.prod.yml up -d

# View logs
docker compose -f docker-compose.prod.yml logs -f app
```

**Simple deploy script** (`deploy.sh`):

```bash
#!/bin/bash
set -e

echo "Pulling latest image..."
docker compose -f docker-compose.prod.yml pull

echo "Restarting services..."
docker compose -f docker-compose.prod.yml up -d

echo "Deployment complete!"
docker compose -f docker-compose.prod.yml ps
```

Then just run: `./deploy.sh`

## Common Operations

### View logs
```bash
docker compose -f docker-compose.prod.yml logs -f
docker compose -f docker-compose.prod.yml logs -f app  # Just app logs
```

### Check status
```bash
docker compose -f docker-compose.prod.yml ps
```

### Restart services
```bash
docker compose -f docker-compose.prod.yml restart app
```

### Database backup
```bash
# Create backup
docker exec aletheia-db pg_dump -U postgres aletheia > backup_$(date +%Y%m%d).sql

# Restore
docker exec -i aletheia-db psql -U postgres aletheia < backup.sql
```

### Access database
```bash
docker exec -it aletheia-db psql -U postgres -d aletheia
```

### Update config
```bash
# Edit .env file
nano .env

# Restart to apply
docker compose -f docker-compose.prod.yml up -d
```

## Rollback

Use a specific version:
```bash
# Edit docker-compose.prod.yml
# Change: image: yourusername/aletheia:latest
# To:     image: yourusername/aletheia:1.0.0

docker compose -f docker-compose.prod.yml up -d
```

## Cost & Services

**Free tier usage:**
- GitHub Actions: Free for public repos (2000 min/month for private)
- Docker Hub: Free (unlimited public images, 1 private repo)
- Your VPS: Whatever you're paying for hosting

**No additional services needed** - that's it!

## Notes

- **Manual deployment**: You control when to deploy (no auto-updates)
- **Simple rollback**: Just change image tag
- **Local uploads**: Uses Docker volume for MVP (switch to S3 when ready)
- **Database**: Also in Docker volume (backup regularly)
