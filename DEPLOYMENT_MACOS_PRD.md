# DFS Lineup Optimizer - macOS Production Deployment PRD

## Executive Summary

This document outlines the comprehensive plan for deploying the DFS Lineup Optimizer application on a macOS server as a production environment. The deployment strategy focuses on reliability, security, performance, and cost-effectiveness while leveraging the existing Docker-based architecture and free API ecosystem.

### Key Objectives
- Deploy a production-ready DFS optimizer on macOS hardware
- Ensure 99.9% uptime with proper monitoring and alerting
- Implement security best practices for handling user data
- Optimize for performance with limited free API constraints
- Enable zero-downtime deployments and easy rollbacks
- Minimize operational costs while maximizing reliability

## Technical Architecture

### System Overview
```
┌─────────────────────────────────────────────────────────────┐
│                    macOS Production Server                   │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │   Nginx     │────│   Frontend   │    │   Backend    │  │
│  │  (Reverse   │    │  (React App) │    │  (Go API)    │  │
│  │   Proxy)    │    │   Port 3000  │    │  Port 8080   │  │
│  └──────┬──────┘    └──────────────┘    └──────┬───────┘  │
│         │                                        │          │
│  ┌──────┴───────────────────────────────────────┴───────┐  │
│  │                   Docker Network                      │  │
│  └──────┬────────────────────┬──────────────────────────┘  │
│  ┌──────┴──────┐      ┌──────┴──────┐                     │
│  │  PostgreSQL │      │    Redis    │                     │
│  │   Database  │      │    Cache    │                     │
│  └─────────────┘      └─────────────┘                     │
└─────────────────────────────────────────────────────────────┘
```

### Component Architecture

#### 1. **Load Balancer / Reverse Proxy**
- **Technology**: Nginx
- **Purpose**: SSL termination, request routing, static file serving
- **Configuration**:
  - SSL/TLS with Let's Encrypt certificates
  - HTTP/2 enabled
  - Gzip compression
  - Rate limiting per IP
  - Security headers (HSTS, CSP, etc.)

#### 2. **Application Containers**
- **Frontend**: React application served via Node.js
- **Backend**: Go API server with WebSocket support
- **Database**: PostgreSQL 15 with connection pooling
- **Cache**: Redis 7 for API response caching
- **Container Orchestration**: Docker Compose with health checks

#### 3. **Data Layer**
- **PostgreSQL**: Primary data store with replication
- **Redis**: Caching layer for API responses and session data
- **File Storage**: Local filesystem for CSV exports
- **Backup**: Automated daily backups to external storage

## Infrastructure Requirements

### Hardware Specifications
- **Minimum Requirements**:
  - macOS 12.0 (Monterey) or later
  - Apple Silicon (M1/M2) or Intel processor
  - 16GB RAM minimum (32GB recommended)
  - 500GB SSD storage
  - Gigabit Ethernet connection
  - UPS for power protection

### Software Requirements
- **Operating System**: macOS 12.0+
- **Runtime**: Docker Desktop for Mac
- **Development Tools**: Xcode Command Line Tools
- **Monitoring**: Prometheus, Grafana, Loki
- **Backup**: Time Machine + cloud backup solution

### Network Configuration
- **Static IP**: Required for production
- **Firewall Rules**:
  - Port 80 (HTTP) - Redirect to HTTPS
  - Port 443 (HTTPS) - Main application
  - Port 22 (SSH) - Management (restricted)
  - Port 3000 (Grafana) - Monitoring (restricted)

## Security Specifications

### Application Security
1. **Authentication & Authorization**
   - JWT-based authentication with refresh tokens
   - Role-based access control (RBAC)
   - Session timeout after 24 hours
   - Password complexity requirements
   - Account lockout after failed attempts

2. **Data Protection**
   - All data encrypted in transit (TLS 1.3)
   - Database encryption at rest
   - API key rotation every 90 days
   - PII data masking in logs
   - GDPR compliance measures

3. **Network Security**
   - Web Application Firewall (WAF) rules
   - DDoS protection via Cloudflare
   - IP whitelisting for admin access
   - Regular security updates
   - Intrusion detection system

### API Security
- **Rate Limiting**: Per-user and per-IP limits
- **API Keys**: Encrypted storage, regular rotation
- **CORS**: Strict origin validation
- **Input Validation**: All inputs sanitized
- **SQL Injection**: Prepared statements only

## Performance Optimization

### Caching Strategy
```yaml
cache_layers:
  - level: CDN
    items: [static_assets, images, fonts]
    ttl: 7_days
  
  - level: nginx
    items: [api_responses, player_data]
    ttl: 1_hour
  
  - level: redis
    items: [session_data, optimization_results]
    ttl: 30_minutes
  
  - level: application
    items: [projections, correlations]
    ttl: 5_minutes
```

### Database Optimization
- **Indexes**: On frequently queried columns
- **Connection Pooling**: PgBouncer configuration
- **Query Optimization**: EXPLAIN ANALYZE on slow queries
- **Partitioning**: Historical data by date
- **Vacuum Schedule**: Daily during off-peak hours

### API Rate Limit Management
```javascript
rateLimits: {
  espn: { requests: 1000, window: '1h', strategy: 'sliding' },
  balldontlie: { requests: 5, window: '1m', strategy: 'fixed' },
  thesportsdb: { requests: 100, window: '1h', strategy: 'sliding' },
  mysportsfeeds: { requests: 500, window: '24h', strategy: 'fixed' }
}
```

## Deployment Configuration

### Docker Compose Production Setup
```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  nginx:
    image: nginx:alpine
    container_name: dfs_nginx
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf
      - ./nginx/ssl:/etc/nginx/ssl
      - ./frontend/dist:/usr/share/nginx/html
    depends_on:
      - backend
      - frontend
    restart: always
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  postgres:
    image: postgres:15-alpine
    container_name: dfs_postgres
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: dfs_optimizer
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./backups:/backups
    ports:
      - "127.0.0.1:5432:5432"
    restart: always
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 30s
      timeout: 10s
      retries: 5
    command: 
      - "postgres"
      - "-c"
      - "max_connections=200"
      - "-c"
      - "shared_buffers=256MB"

  redis:
    image: redis:7-alpine
    container_name: dfs_redis
    command: redis-server --appendonly yes --maxmemory 2gb --maxmemory-policy allkeys-lru
    volumes:
      - redis_data:/data
    ports:
      - "127.0.0.1:6379:6379"
    restart: always
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 30s
      timeout: 10s
      retries: 5

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile.prod
    container_name: dfs_backend
    environment:
      PORT: 8080
      ENV: production
      DATABASE_URL: postgres://${DB_USER}:${DB_PASSWORD}@postgres:5432/dfs_optimizer?sslmode=require
      REDIS_URL: redis://redis:6379/0
      JWT_SECRET: ${JWT_SECRET}
      CORS_ORIGINS: ${CORS_ORIGINS}
      BALLDONTLIE_API_KEY: ${BALLDONTLIE_API_KEY}
      THESPORTSDB_API_KEY: ${THESPORTSDB_API_KEY}
      LOG_LEVEL: info
      METRICS_ENABLED: true
    ports:
      - "127.0.0.1:8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: always
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile.prod
      args:
        VITE_API_URL: ${FRONTEND_API_URL}
    container_name: dfs_frontend
    ports:
      - "127.0.0.1:3000:3000"
    depends_on:
      - backend
    restart: always
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:3000"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  postgres_data:
    driver: local
  redis_data:
    driver: local
```

### Environment Configuration
```bash
# .env.production
# Server Configuration
PORT=8080
ENV=production
NODE_ENV=production

# Database Configuration
DB_USER=dfs_admin
DB_PASSWORD=<strong-password>
DATABASE_URL=postgres://dfs_admin:<password>@localhost:5432/dfs_optimizer?sslmode=require

# Redis Configuration
REDIS_URL=redis://localhost:6379/0
REDIS_PASSWORD=<redis-password>

# JWT Configuration
JWT_SECRET=<64-character-secret>
JWT_EXPIRY=24h
REFRESH_TOKEN_EXPIRY=7d

# CORS Configuration
CORS_ORIGINS=https://yourdomain.com,https://www.yourdomain.com
CORS_CREDENTIALS=true

# API Keys (Encrypted)
BALLDONTLIE_API_KEY=<encrypted-key>
THESPORTSDB_API_KEY=<encrypted-key>
ENCRYPTION_KEY=<32-character-key>

# Optimization Settings
MAX_LINEUPS=150
OPTIMIZATION_TIMEOUT=30
MAX_CONCURRENT_OPTIMIZATIONS=10

# Simulation Settings
MAX_SIMULATIONS=100000
SIMULATION_WORKERS=8
SIMULATION_CACHE_TTL=300

# Monitoring
METRICS_ENABLED=true
METRICS_PORT=9090
LOG_LEVEL=info
SENTRY_DSN=<sentry-dsn>

# Rate Limiting
RATE_LIMIT_WINDOW=60
RATE_LIMIT_MAX_REQUESTS=100
RATE_LIMIT_SKIP_SUCCESSFUL_REQUESTS=false

# Frontend
FRONTEND_API_URL=https://api.yourdomain.com/v1
```

## Monitoring & Observability

### Metrics Collection
1. **Application Metrics**
   - Request rate and latency
   - Error rates by endpoint
   - Active users and sessions
   - Optimization queue length
   - API usage by provider

2. **System Metrics**
   - CPU and memory usage
   - Disk I/O and space
   - Network throughput
   - Container health status
   - Database connections

3. **Business Metrics**
   - Daily active users
   - Lineups generated
   - API cost tracking
   - Feature usage analytics
   - User retention

### Logging Strategy
```yaml
logging:
  levels:
    production: [error, warn, info]
    staging: [error, warn, info, debug]
  
  destinations:
    - type: stdout
      format: json
      fields: [timestamp, level, message, context]
    
    - type: file
      path: /var/log/dfs-optimizer/
      rotation: daily
      retention: 30_days
    
    - type: loki
      endpoint: http://localhost:3100
      labels: [app, env, version]
```

### Alerting Rules
```yaml
alerts:
  - name: high_error_rate
    condition: error_rate > 5%
    duration: 5m
    severity: critical
    
  - name: api_rate_limit_approaching
    condition: api_usage > 80%
    duration: 10m
    severity: warning
    
  - name: database_connection_pool_exhausted
    condition: available_connections < 10
    duration: 5m
    severity: critical
    
  - name: optimization_timeout
    condition: optimization_duration > 30s
    duration: 1m
    severity: warning
```

## Backup & Disaster Recovery

### Backup Strategy
1. **Database Backups**
   - Full backup: Daily at 2 AM
   - Incremental: Every 6 hours
   - Retention: 30 days local, 90 days cloud
   - Test restore: Weekly

2. **Application State**
   - Redis snapshots: Every hour
   - Configuration files: Git repository
   - User uploads: S3 compatible storage
   - Encryption: AES-256

3. **Backup Locations**
   - Primary: Local Time Machine
   - Secondary: AWS S3 or Backblaze B2
   - Tertiary: Offsite physical backup

### Recovery Procedures
```bash
# Database Recovery
1. Stop application containers
2. Restore PostgreSQL from backup
3. Verify data integrity
4. Start application containers
5. Run smoke tests

# Full System Recovery
1. Provision new macOS server
2. Install Docker Desktop
3. Clone repository
4. Restore data from backups
5. Update DNS records
6. Verify all services
```

## CI/CD Pipeline

### GitHub Actions Workflow
```yaml
name: Deploy to Production

on:
  push:
    tags:
      - 'v*'

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run tests
        run: |
          docker-compose -f docker-compose.test.yml up --abort-on-container-exit
          
  build:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Build images
        run: |
          docker build -t dfs-backend:${{ github.ref_name }} ./backend
          docker build -t dfs-frontend:${{ github.ref_name }} ./frontend
          
  deploy:
    needs: build
    runs-on: self-hosted
    steps:
      - name: Deploy to production
        run: |
          ./scripts/deploy.sh ${{ github.ref_name }}
```

### Deployment Script
```bash
#!/bin/bash
# scripts/deploy.sh

VERSION=$1
BACKUP_DIR="/backups/pre-deploy/$(date +%Y%m%d_%H%M%S)"

# Create pre-deployment backup
echo "Creating backup..."
mkdir -p $BACKUP_DIR
docker exec dfs_postgres pg_dump -U postgres dfs_optimizer > $BACKUP_DIR/database.sql

# Pull new images
echo "Pulling new images..."
docker pull dfs-backend:$VERSION
docker pull dfs-frontend:$VERSION

# Rolling update
echo "Performing rolling update..."
docker-compose -f docker-compose.prod.yml up -d --no-deps backend
sleep 30
docker-compose -f docker-compose.prod.yml up -d --no-deps frontend

# Health check
echo "Running health checks..."
./scripts/health-check.sh

# Cleanup old images
docker image prune -f

echo "Deployment complete!"
```

## Scaling Strategy

### Horizontal Scaling Options
1. **Load Balancer**: HAProxy or Nginx Plus
2. **Backend Replicas**: Up to 3 instances
3. **Read Replicas**: PostgreSQL streaming replication
4. **Cache Cluster**: Redis Sentinel configuration

### Vertical Scaling Triggers
- CPU usage > 80% for 5 minutes
- Memory usage > 85%
- Request queue depth > 100
- Response time p95 > 2 seconds

## Cost Analysis

### Monthly Operating Costs
```
Infrastructure:
- macOS Server (owned): $0
- Electricity: ~$50
- Internet (Business): $150
- Backup Storage (1TB): $20
- Domain & SSL: $15
- Monitoring (Datadog): $0 (free tier)
- Error Tracking (Sentry): $0 (free tier)

Total: ~$235/month

API Costs: $0 (all free tier)
- ESPN API: Free
- BALLDONTLIE: Free tier
- TheSportsDB: Free tier
- MySportsFeeds: Free tier
```

### Cost Optimization Strategies
1. Aggressive caching to minimize API calls
2. Batch processing during off-peak hours
3. Compress all responses
4. Use CDN for static assets
5. Implement request coalescing

## Deployment Guide

### Prerequisites Checklist
- [ ] macOS server provisioned
- [ ] Static IP configured
- [ ] Domain name registered
- [ ] Docker Desktop installed
- [ ] SSL certificates obtained
- [ ] Backup storage configured
- [ ] Monitoring tools installed

### Step-by-Step Deployment

#### 1. Server Preparation
```bash
# Install Homebrew
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install required tools
brew install git docker docker-compose nginx certbot postgres redis

# Clone repository
git clone https://github.com/yourusername/dfs-optimizer.git
cd dfs-optimizer
```

#### 2. SSL Certificate Setup
```bash
# Generate SSL certificates
sudo certbot certonly --standalone -d yourdomain.com -d www.yourdomain.com

# Create certificate renewal cron job
echo "0 0 * * 0 /usr/local/bin/certbot renew --quiet" | crontab -
```

#### 3. Environment Configuration
```bash
# Copy production environment template
cp .env.production.template .env.production

# Generate secure secrets
openssl rand -base64 64 | tr -d '\n' > jwt_secret.txt
openssl rand -base64 32 | tr -d '\n' > encryption_key.txt

# Edit environment variables
nano .env.production
```

#### 4. Database Initialization
```bash
# Start only PostgreSQL
docker-compose -f docker-compose.prod.yml up -d postgres

# Wait for PostgreSQL to be ready
sleep 10

# Run migrations
docker exec dfs_postgres psql -U postgres -d dfs_optimizer -f /migrations/init.sql

# Create application user
docker exec dfs_postgres psql -U postgres -c "CREATE USER dfs_admin WITH ENCRYPTED PASSWORD 'your-password';"
docker exec dfs_postgres psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE dfs_optimizer TO dfs_admin;"
```

#### 5. Application Deployment
```bash
# Build production images
docker-compose -f docker-compose.prod.yml build

# Start all services
docker-compose -f docker-compose.prod.yml up -d

# Check service health
docker-compose -f docker-compose.prod.yml ps

# View logs
docker-compose -f docker-compose.prod.yml logs -f
```

#### 6. Nginx Configuration
```nginx
# /usr/local/etc/nginx/nginx.conf
server {
    listen 80;
    server_name yourdomain.com www.yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com www.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;
    
    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Frontend
    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }

    # Backend API
    location /api {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket support
    location /ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

#### 7. Monitoring Setup
```bash
# Install Prometheus
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  -v ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus

# Install Grafana
docker run -d \
  --name grafana \
  -p 3000:3000 \
  -e "GF_SECURITY_ADMIN_PASSWORD=admin" \
  grafana/grafana

# Import dashboards
# Navigate to http://localhost:3000
# Import dashboard IDs: 1860 (Node Exporter), 893 (Docker), 9628 (PostgreSQL)
```

#### 8. Launch Services
```bash
# Create LaunchDaemon for auto-start
sudo tee /Library/LaunchDaemons/com.dfs-optimizer.plist << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.dfs-optimizer</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/docker-compose</string>
        <string>-f</string>
        <string>/path/to/dfs-optimizer/docker-compose.prod.yml</string>
        <string>up</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardErrorPath</key>
    <string>/var/log/dfs-optimizer.err</string>
    <key>StandardOutPath</key>
    <string>/var/log/dfs-optimizer.log</string>
</dict>
</plist>
EOF

# Load the service
sudo launchctl load /Library/LaunchDaemons/com.dfs-optimizer.plist
```

## Maintenance Procedures

### Daily Tasks
1. Check application health dashboard
2. Review error logs for anomalies
3. Verify backup completion
4. Monitor API usage limits

### Weekly Tasks
1. Review performance metrics
2. Update dependencies if needed
3. Test backup restoration
4. Clear old log files

### Monthly Tasks
1. Security updates
2. Database optimization
3. SSL certificate renewal check
4. Cost analysis review

### Maintenance Scripts
```bash
# scripts/maintenance.sh
#!/bin/bash

echo "Starting maintenance tasks..."

# Cleanup old logs
find /var/log/dfs-optimizer -name "*.log" -mtime +30 -delete

# Vacuum PostgreSQL
docker exec dfs_postgres psql -U postgres -d dfs_optimizer -c "VACUUM ANALYZE;"

# Clear Redis expired keys
docker exec dfs_redis redis-cli EVAL "return redis.call('del', unpack(redis.call('keys', 'PHPREDIS_SESSION:*')))" 0

# Update container images
docker-compose -f docker-compose.prod.yml pull

echo "Maintenance complete!"
```

## Troubleshooting Guide

### Common Issues

#### 1. High Memory Usage
```bash
# Check memory usage
docker stats

# Increase Redis memory limit
docker exec dfs_redis redis-cli CONFIG SET maxmemory 4gb

# Restart containers
docker-compose -f docker-compose.prod.yml restart
```

#### 2. Slow API Response
```bash
# Check slow queries
docker exec dfs_postgres psql -U postgres -d dfs_optimizer -c "SELECT * FROM pg_stat_statements ORDER BY total_time DESC LIMIT 10;"

# Add missing index
docker exec dfs_postgres psql -U postgres -d dfs_optimizer -c "CREATE INDEX idx_player_stats_date ON player_stats(game_date);"
```

#### 3. Rate Limit Errors
```bash
# Check current API usage
docker exec dfs_backend curl http://localhost:8080/admin/api-usage

# Increase cache TTL
docker exec dfs_redis redis-cli CONFIG SET maxmemory-policy allkeys-lru
```

### Health Check Endpoints
- **Application**: `https://yourdomain.com/api/health`
- **Database**: `https://yourdomain.com/api/health/db`
- **Redis**: `https://yourdomain.com/api/health/redis`
- **APIs**: `https://yourdomain.com/api/health/external`

## Security Checklist

- [ ] Change all default passwords
- [ ] Enable macOS firewall
- [ ] Configure fail2ban for SSH
- [ ] Set up VPN for admin access
- [ ] Enable audit logging
- [ ] Implement backup encryption
- [ ] Regular security scans
- [ ] Incident response plan

## Performance Benchmarks

### Target Metrics
- **API Response Time**: < 200ms (p95)
- **Optimization Time**: < 10s for 150 lineups
- **Concurrent Users**: 100 simultaneous
- **Uptime**: 99.9% monthly
- **Error Rate**: < 0.1%

### Load Testing
```bash
# Install Apache Bench
brew install apache-bench

# Test API endpoint
ab -n 1000 -c 10 https://yourdomain.com/api/v1/players

# Test optimization endpoint
ab -n 100 -c 5 -p optimization.json -T application/json https://yourdomain.com/api/v1/optimize
```

## Conclusion

This PRD provides a comprehensive guide for deploying the DFS Lineup Optimizer on a macOS production server. The architecture prioritizes reliability, security, and performance while maintaining zero API costs through intelligent caching and rate limit management.

Key success factors:
1. Robust monitoring and alerting
2. Automated backup and recovery
3. Security-first approach
4. Performance optimization
5. Clear operational procedures

Following this guide will result in a production-ready deployment capable of serving hundreds of users while maintaining high availability and performance standards.