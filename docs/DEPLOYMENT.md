# Flowra Deployment Guide

This guide covers deploying Flowra in various environments, from local development to production.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Configuration](#configuration)
4. [Docker Compose Deployment](#docker-compose-deployment)
5. [Manual Deployment](#manual-deployment)
6. [Environment Variables](#environment-variables)
7. [Health Checks](#health-checks)
8. [Monitoring](#monitoring)
9. [Troubleshooting](#troubleshooting)
10. [Security Considerations](#security-considerations)

---

## Prerequisites

### Required Software

| Software | Minimum Version | Purpose |
|----------|----------------|---------|
| Go | 1.25+ | Backend runtime |
| Docker | 24.0+ | Container runtime |
| Docker Compose | 2.20+ | Multi-container orchestration |
| MongoDB | 6.0+ | Primary database |
| Redis | 7.0+ | Cache and pub/sub |
| Keycloak | 23.0+ | Authentication server |

### System Requirements

| Environment | CPU | RAM | Storage |
|-------------|-----|-----|---------|
| Development | 2 cores | 4 GB | 10 GB |
| Staging | 4 cores | 8 GB | 50 GB |
| Production | 8+ cores | 16+ GB | 100+ GB |

### Network Requirements

- Port 8080: API server
- Port 8090: Keycloak admin console
- Port 27017: MongoDB
- Port 6379: Redis

---

## Quick Start

### 1. Clone and Configure

```bash
# Clone the repository
git clone https://github.com/lllypuk/flowra.git
cd flowra

# Copy example configuration
cp configs/config.example.yaml configs/config.yaml

# Edit configuration as needed
vim configs/config.yaml
```

### 2. Start Infrastructure

```bash
# Start all services with Docker Compose
docker-compose up -d

# Verify services are running
docker-compose ps
```

### 3. Initialize Database

```bash
# Run database migrations
make migrate-up

# Or manually
go run cmd/migrator/main.go up
```

### 4. Start Application

```bash
# Development mode
make dev

# Or production build
make build
./bin/api
```

### 5. Verify Deployment

```bash
# Check health endpoint
curl http://localhost:8080/health

# Check readiness
curl http://localhost:8080/ready
```

---

## Configuration

### Configuration File

The main configuration file is `configs/config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  shutdown_timeout: 10s

mongodb:
  uri: "mongodb://admin:admin123@localhost:27017"
  database: "flowra"
  timeout: 10s
  max_pool_size: 100

redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  pool_size: 10

keycloak:
  url: "http://localhost:8090"
  realm: "flowra"
  client_id: "flowra-backend"
  client_secret: "your-client-secret"
  admin_username: "admin"
  admin_password: "admin123"

auth:
  jwt_secret: "your-secure-jwt-secret"
  access_token_ttl: 15m
  refresh_token_ttl: 7d

eventbus:
  type: "redis"  # redis | inmemory
  redis_channel_prefix: "events."

log:
  level: "info"  # debug | info | warn | error
  format: "json"  # json | text

websocket:
  read_buffer_size: 1024
  write_buffer_size: 1024
  ping_interval: 30s
  pong_timeout: 60s
```

### Configuration Precedence

1. Environment variables (highest priority)
2. Configuration file
3. Default values (lowest priority)

---

## Docker Compose Deployment

### Development Environment

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### Production Environment

Create `docker-compose.prod.yml`:

```yaml
version: "3.8"

services:
  api:
    image: flowra/api:latest
    ports:
      - "8080:8080"
    environment:
      - FLOWRA_ENV=production
      - FLOWRA_MONGODB_URI=mongodb://mongodb:27017
      - FLOWRA_REDIS_ADDR=redis:6379
    depends_on:
      mongodb:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    restart: unless-stopped
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G

  mongodb:
    image: mongo:6.0
    volumes:
      - mongodb_data:/data/db
    environment:
      MONGO_INITDB_ROOT_USERNAME: ${MONGO_USER}
      MONGO_INITDB_ROOT_PASSWORD: ${MONGO_PASSWORD}
    healthcheck:
      test: ["CMD", "mongosh", "--eval", "db.adminCommand('ping')"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "${REDIS_PASSWORD}", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  keycloak:
    image: quay.io/keycloak/keycloak:23.0
    command: start
    environment:
      KC_DB: postgres
      KC_DB_URL: jdbc:postgresql://keycloak-db:5432/keycloak
      KC_DB_USERNAME: ${KC_DB_USER}
      KC_DB_PASSWORD: ${KC_DB_PASSWORD}
      KC_HOSTNAME: auth.flowra.com
      KEYCLOAK_ADMIN: ${KEYCLOAK_ADMIN}
      KEYCLOAK_ADMIN_PASSWORD: ${KEYCLOAK_ADMIN_PASSWORD}
    depends_on:
      - keycloak-db
    restart: unless-stopped

  keycloak-db:
    image: postgres:15-alpine
    volumes:
      - keycloak_db_data:/var/lib/postgresql/data
    environment:
      POSTGRES_DB: keycloak
      POSTGRES_USER: ${KC_DB_USER}
      POSTGRES_PASSWORD: ${KC_DB_PASSWORD}
    restart: unless-stopped

volumes:
  mongodb_data:
  redis_data:
  keycloak_db_data:
```

### Start Production

```bash
# Create .env file with production values
cat > .env << EOF
MONGO_USER=flowra
MONGO_PASSWORD=<secure-password>
REDIS_PASSWORD=<secure-password>
KC_DB_USER=keycloak
KC_DB_PASSWORD=<secure-password>
KEYCLOAK_ADMIN=admin
KEYCLOAK_ADMIN_PASSWORD=<secure-password>
EOF

# Start production stack
docker-compose -f docker-compose.prod.yml up -d
```

---

## Manual Deployment

### Build Application

```bash
# Build all binaries
make build

# This creates:
# - bin/api        (API server)
# - bin/worker     (Background worker)
# - bin/migrator   (Database migrations)
```

### Run Components

```bash
# 1. Run migrations first
./bin/migrator up

# 2. Start API server
./bin/api

# 3. Start worker (separate terminal/process)
./bin/worker
```

### Systemd Service

Create `/etc/systemd/system/flowra-api.service`:

```ini
[Unit]
Description=Flowra API Server
After=network.target mongodb.service redis.service

[Service]
Type=simple
User=flowra
Group=flowra
WorkingDirectory=/opt/flowra
ExecStart=/opt/flowra/bin/api
Restart=always
RestartSec=5
Environment=FLOWRA_ENV=production
EnvironmentFile=/opt/flowra/.env

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/flowra/logs

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable flowra-api
sudo systemctl start flowra-api
sudo systemctl status flowra-api
```

---

## Environment Variables

All configuration options can be overridden via environment variables using the `FLOWRA_` prefix.

### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `FLOWRA_SERVER_HOST` | `0.0.0.0` | Server bind address |
| `FLOWRA_SERVER_PORT` | `8080` | Server port |
| `FLOWRA_SERVER_READ_TIMEOUT` | `30s` | Request read timeout |
| `FLOWRA_SERVER_WRITE_TIMEOUT` | `30s` | Response write timeout |
| `FLOWRA_SERVER_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout |

### Database Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `FLOWRA_MONGODB_URI` | `mongodb://localhost:27017` | MongoDB connection URI |
| `FLOWRA_MONGODB_DATABASE` | `flowra` | Database name |
| `FLOWRA_MONGODB_TIMEOUT` | `10s` | Connection timeout |
| `FLOWRA_MONGODB_MAX_POOL_SIZE` | `100` | Max connection pool size |

### Redis Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `FLOWRA_REDIS_ADDR` | `localhost:6379` | Redis address |
| `FLOWRA_REDIS_PASSWORD` | `` | Redis password |
| `FLOWRA_REDIS_DB` | `0` | Redis database number |
| `FLOWRA_REDIS_POOL_SIZE` | `10` | Connection pool size |

### Authentication Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `FLOWRA_KEYCLOAK_URL` | `http://localhost:8090` | Keycloak URL |
| `FLOWRA_KEYCLOAK_REALM` | `flowra` | Keycloak realm |
| `FLOWRA_KEYCLOAK_CLIENT_ID` | `flowra-backend` | OAuth client ID |
| `FLOWRA_KEYCLOAK_CLIENT_SECRET` | `` | OAuth client secret |
| `FLOWRA_AUTH_JWT_SECRET` | `` | JWT signing secret |
| `FLOWRA_AUTH_ACCESS_TOKEN_TTL` | `15m` | Access token lifetime |
| `FLOWRA_AUTH_REFRESH_TOKEN_TTL` | `7d` | Refresh token lifetime |

### Logging Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `FLOWRA_LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |
| `FLOWRA_LOG_FORMAT` | `json` | Log format (json/text) |
| `FLOWRA_ENV` | `development` | Environment name |

### WebSocket Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `FLOWRA_WEBSOCKET_READ_BUFFER_SIZE` | `1024` | Read buffer size |
| `FLOWRA_WEBSOCKET_WRITE_BUFFER_SIZE` | `1024` | Write buffer size |
| `FLOWRA_WEBSOCKET_PING_INTERVAL` | `30s` | Ping interval |
| `FLOWRA_WEBSOCKET_PONG_TIMEOUT` | `60s` | Pong timeout |

---

## Health Checks

### Endpoints

| Endpoint | Purpose | Response |
|----------|---------|----------|
| `GET /health` | Liveness probe | `{"status": "healthy"}` |
| `GET /ready` | Readiness probe | `{"status": "ready", "components": {...}}` |
| `GET /health/details` | Detailed health | Full component status |

### Kubernetes Probes

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
```

### Docker Health Check

```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 30s
```

---

## Monitoring

### Metrics

Flowra exposes Prometheus-compatible metrics at `/metrics` (when enabled):

- `flowra_http_requests_total` - Total HTTP requests
- `flowra_http_request_duration_seconds` - Request latency histogram
- `flowra_websocket_connections` - Active WebSocket connections
- `flowra_mongodb_operations_total` - MongoDB operations
- `flowra_redis_operations_total` - Redis operations

### Logging

Structured JSON logging is used by default in production:

```json
{
  "time": "2026-01-28T10:30:00Z",
  "level": "INFO",
  "msg": "HTTP request",
  "method": "GET",
  "path": "/api/v1/workspaces",
  "status": 200,
  "duration_ms": 45,
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### Recommended Stack

- **Prometheus** - Metrics collection
- **Grafana** - Visualization and dashboards
- **Loki** - Log aggregation
- **Jaeger** - Distributed tracing (future)

---

## Troubleshooting

### Common Issues

#### 1. Cannot Connect to MongoDB

**Symptoms:**
```
error: failed to connect to MongoDB: connection refused
```

**Solutions:**
```bash
# Check MongoDB is running
docker-compose ps mongodb

# Check MongoDB logs
docker-compose logs mongodb

# Test connection
mongosh "mongodb://admin:admin123@localhost:27017"

# Verify network
docker network ls
docker network inspect flowra-network
```

#### 2. Cannot Connect to Redis

**Symptoms:**
```
error: failed to connect to Redis: connection refused
```

**Solutions:**
```bash
# Check Redis is running
docker-compose ps redis

# Test connection
redis-cli -h localhost -p 6379 ping

# Check Redis logs
docker-compose logs redis
```

#### 3. Keycloak Authentication Fails

**Symptoms:**
```
error: invalid token or token expired
```

**Solutions:**
```bash
# Check Keycloak is accessible
curl http://localhost:8090/realms/flowra

# Verify realm exists
# Login to Keycloak admin: http://localhost:8090/admin

# Check client configuration
# Ensure client_secret matches in config

# Check time synchronization
timedatectl status
```

#### 4. WebSocket Connection Fails

**Symptoms:**
- WebSocket upgrade fails
- Connection drops immediately

**Solutions:**
```bash
# Check if WebSocket endpoint is accessible
curl -i -N -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Key: test" \
  -H "Sec-WebSocket-Version: 13" \
  http://localhost:8080/api/v1/ws

# Check for proxy issues (nginx/traefik)
# Ensure WebSocket upgrade headers are passed through

# Check authentication token
# Token must be valid and not expired
```

#### 5. High Memory Usage

**Symptoms:**
- API server consuming excessive memory
- OOM kills in production

**Solutions:**
```bash
# Check current memory usage
docker stats

# Analyze Go memory profile
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Tune connection pools
# Reduce FLOWRA_MONGODB_MAX_POOL_SIZE
# Reduce FLOWRA_REDIS_POOL_SIZE
```

#### 6. Slow API Responses

**Symptoms:**
- High latency on API calls
- Timeouts

**Solutions:**
```bash
# Check MongoDB indexes
mongosh flowra --eval "db.chats.getIndexes()"

# Analyze slow queries
mongosh flowra --eval "db.setProfilingLevel(1, { slowms: 100 })"

# Check Redis latency
redis-cli --latency

# Review request logs
grep "duration_ms" /var/log/flowra/api.log | sort -t: -k2 -n | tail -20
```

### Debug Mode

Enable debug logging for detailed troubleshooting:

```bash
# Via environment variable
export FLOWRA_LOG_LEVEL=debug
./bin/api

# Or in config.yaml
log:
  level: "debug"
```

### Log Locations

| Component | Location |
|-----------|----------|
| API Server | `stdout` / `/var/log/flowra/api.log` |
| Worker | `stdout` / `/var/log/flowra/worker.log` |
| MongoDB | Docker logs / `/var/log/mongodb/` |
| Redis | Docker logs / `/var/log/redis/` |
| Keycloak | Docker logs |

---

## Security Considerations

### Production Checklist

- [ ] Change all default passwords
- [ ] Use strong JWT secret (min 256 bits)
- [ ] Enable TLS/HTTPS
- [ ] Configure firewall rules
- [ ] Enable MongoDB authentication
- [ ] Enable Redis password
- [ ] Configure CORS properly
- [ ] Set up rate limiting
- [ ] Enable audit logging
- [ ] Regular security updates

### TLS Configuration

For production, use a reverse proxy (nginx, traefik) with TLS:

```nginx
server {
    listen 443 ssl http2;
    server_name api.flowra.com;

    ssl_certificate /etc/letsencrypt/live/api.flowra.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.flowra.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Secrets Management

For production, use a secrets manager:

- HashiCorp Vault
- AWS Secrets Manager
- Azure Key Vault
- Kubernetes Secrets

---

## Backup and Recovery

### MongoDB Backup

```bash
# Create backup
mongodump --uri="mongodb://admin:password@localhost:27017" --out=/backup/$(date +%Y%m%d)

# Restore backup
mongorestore --uri="mongodb://admin:password@localhost:27017" /backup/20260128
```

### Redis Backup

```bash
# Redis automatically persists to disk (AOF/RDB)
# Copy persistence files for backup
cp /var/lib/redis/dump.rdb /backup/redis/dump.rdb.$(date +%Y%m%d)
```

---

*Last updated: January 2026*
