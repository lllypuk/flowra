# Flowra Deployment Guide

This guide covers deploying Flowra in various environments, from local development to production.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Configuration](#configuration)
4. [Docker (self-hosted)](#docker-self-hosted)
5. [Manual Deployment](#manual-deployment)
6. [Environment Variables](#environment-variables)
7. [Health Checks](#health-checks)
8. [Monitoring](#monitoring)
9. [Troubleshooting](#troubleshooting)
10. [Security Considerations](#security-considerations)
11. [Always-on install](#always-on-install-systemd-user-units)

---

## Prerequisites

### Required Software

| Software | Minimum Version | Purpose |
|----------|----------------|---------|
| Go | 1.26+ | Backend runtime |
| Docker | 24.0+ | Container runtime |
| Docker Compose | 2.20+ | Multi-container orchestration |
| MongoDB | 6.0+ | Primary database |
| Redis | 7.0+ | Cache and pub/sub |

### System Requirements

| Environment | CPU | RAM | Storage |
|-------------|-----|-----|---------|
| Development | 2 cores | 4 GB | 10 GB |
| Staging | 4 cores | 8 GB | 50 GB |
| Production | 8+ cores | 16+ GB | 100+ GB |

### Network Requirements

- Port 8080: API server, on loopback only (`127.0.0.1`)
- Port 27017: MongoDB (only if exposed for external admin/debug access)
- Port 6379: Redis (only if exposed for external admin/debug access)

---

## Quick Start

### 1. Clone and Configure

```bash
# Clone the repository
git clone https://github.com/lllypuk/flowra.git
cd flowra

# Generate an owner token and give the server its digest
token=$(openssl rand -hex 32); echo "$token"
export AUTH_OWNER_TOKEN_SHA256=$(printf %s "$token" | sha256sum | cut -d' ' -f1)
```

### 2. Start Infrastructure

```bash
# Start all services with Docker Compose
docker-compose up -d

# Verify services are running
docker-compose ps
```

### 3. Start Application

```bash
# Development mode
make dev

# Or production build
make build
./bin/api
```

### 4. Verify Deployment

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
  host: "127.0.0.1"  # the API refuses a non-loopback address
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

auth:
  # tokens:            # named owner tokens, SHA-256 digests only
  #   - name: "phone"
  #     sha256: "<hex sha256>"
  owner:
    username: "owner"
    email: "owner@flowra.local"
  session_ttl: 720h

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

## Docker (self-hosted)

Use the production compose stack when deploying Flowra as a single image with bundled API+worker runtime.

### Start and Stop

```bash
# 1) Prepare environment variables
cp .env.example .env

# 2) Set required secrets before first run
#    - MONGODB_ROOT_USERNAME (for example flowra) and MONGODB_ROOT_PASSWORD
#    - REDIS_PASSWORD
#    - AUTH_OWNER_TOKEN_SHA256: token=$(openssl rand -hex 32); printf %s "$token" | sha256sum
vim .env

# 3) Start the stack (MongoDB, Redis, Flowra app)
docker compose -f docker-compose.prod.yml up -d

# 4) Verify health
curl -sf http://localhost:8080/health

# 5) View logs
docker compose -f docker-compose.prod.yml logs -f app

# 6) Stop the stack
docker compose -f docker-compose.prod.yml down
```

Equivalent Make targets:

```bash
make docker-build
make docker-prod-up
make docker-prod-logs
make docker-prod-down
```

### Persistent Volumes

`docker-compose.prod.yml` uses named volumes for data persistence:
- `uploads_data` mounted to `/app/uploads` in `app` for user file uploads
- `mongodb_data` mounted to `/data/db` in `mongodb`
- `redis_data` mounted to `/data` in `redis`

Data in these volumes survives `docker compose ... down`. Use `docker compose ... down -v` only when intentionally resetting state.

### Network

- The token is a bearer secret over plain HTTP, so the API listens only on `127.0.0.1`. The `app` service runs
  with `network_mode: host`; MongoDB and Redis are published on `127.0.0.1` only.
- The stack cannot run next to the development stack: both use ports 27017 and 6379.
- Access from other machines (a private network or an HTTPS proxy on the host) is not configured here.

### Migrating a database from the Keycloak era

Users created through Keycloak store its subject in `keycloak_id`; every token now resolves to the identity
`owner`. Point the owner user at it once, before the first start of this version:

```bash
docker compose -f docker-compose.prod.yml up -d mongodb
docker compose -f docker-compose.prod.yml exec mongodb mongosh \
  -u "$MONGODB_ROOT_USERNAME" -p "$MONGODB_ROOT_PASSWORD" --authenticationDatabase admin flowra \
  --eval 'db.users.updateOne({username: "<old owner username>"}, {$set: {keycloak_id: "owner"}})'
```

Without it, the first sign-in looks for an existing user with the `auth.owner.username`: if the old owner has
that username, it is used (with an error logged on every request until the step is done); otherwise a new user
without workspaces is created. If the server already ran and created that user, unset its link first, then run the
step above:

```bash
docker compose -f docker-compose.prod.yml exec mongodb mongosh \
  -u "$MONGODB_ROOT_USERNAME" -p "$MONGODB_ROOT_PASSWORD" --authenticationDatabase admin flowra \
  --eval 'db.users.updateOne({keycloak_id: "owner", username: {$ne: "<old owner username>"}}, {$unset: {keycloak_id: ""}})'
```

Leftover `keycloak_group_id` fields and the `idx_workspaces_keycloak_unique` index on workspaces are harmless.

### Worker Runtime Mode

- Unified mode: set `FLOWRA_WORKER=true` to run API and background worker loops in the same `api` process (enabled by default in `Dockerfile` and `docker-compose.prod.yml`).
- Separate mode: set `FLOWRA_WORKER=false` for API-only process and run `./bin/worker` separately (used in manual/non-single-image deployments).
- CLI override: run `./bin/api --with-worker` (or `--with-worker=false`) to override `FLOWRA_WORKER`.

---

## Manual Deployment

### Build Application

```bash
# Build all binaries
make build

# This creates:
# - bin/api        (API server)
# - bin/worker     (Background worker)
```

### Run Components

```bash
# 1. Start API server
./bin/api

# 2. Start worker (separate terminal/process)
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
Environment=LOG_LEVEL=info
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

Flowra environment variable names do not use a `FLOWRA_` prefix (except runtime toggles like `FLOWRA_WORKER` and `FLOWRA_DEV_MODE`).

### Runtime Toggles

| Variable | Default | Description |
|----------|---------|-------------|
| `FLOWRA_WORKER` | `false` in app config (`true` in Dockerfile/compose) | Run background worker loops inside the API process |
| `FLOWRA_DEV_MODE` | `` | Development mode selector (`lite` for API-only local mode) |

### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_HOST` | `127.0.0.1` | Server bind address; the API refuses a non-loopback address |
| `SERVER_PORT` | `8080` | Server port |
| `SERVER_READ_TIMEOUT` | `30s` | Request read timeout |
| `SERVER_WRITE_TIMEOUT` | `30s` | Response write timeout |
| `SERVER_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout |

### MongoDB Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `MONGODB_URI` | `mongodb://localhost:27017` | MongoDB connection URI |
| `MONGODB_DATABASE` | `flowra` | Database name |
| `MONGODB_TIMEOUT` | `10s` | Connection timeout |
| `MONGODB_MAX_POOL_SIZE` | `100` | Max connection pool size |

### Redis Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `REDIS_PASSWORD` | `` | Redis password |
| `REDIS_DB` | `0` | Redis database number |
| `REDIS_POOL_SIZE` | `10` | Connection pool size |

### Auth Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `AUTH_OWNER_TOKEN_SHA256` | `` | Hex SHA-256 of the owner token named `owner`; more tokens go to `auth.tokens` in YAML |
| `AUTH_OWNER_USERNAME` | `owner` | Username every token resolves to |
| `AUTH_OWNER_EMAIL` | `owner@flowra.local` | Email of the owner user |
| `AUTH_SESSION_TTL` | `720h` | Lifetime of the web session cookie |

### Logging Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |
| `LOG_FORMAT` | `json` | Log format (`json` or `text`) |

### WebSocket Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `WS_READ_BUFFER_SIZE` | `1024` | Read buffer size |
| `WS_WRITE_BUFFER_SIZE` | `1024` | Write buffer size |
| `WS_PING_INTERVAL` | `30s` | Ping interval |
| `WS_PONG_TIMEOUT` | `60s` | Pong timeout |

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

#### 3. Sign-in Fails

**Symptoms:**
```
error: invalid token
```

**Solutions:**
```bash
# The server stores only the digest: compare it with the token you send
printf %s "$token" | sha256sum

# The API refuses to start without a token or on a non-loopback address
docker compose -f docker-compose.prod.yml logs app | grep -i auth
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
# Reduce MONGODB_MAX_POOL_SIZE
# Reduce REDIS_POOL_SIZE
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
export LOG_LEVEL=debug
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

---

## Security Considerations

### Production Checklist

- [ ] Change all default passwords
- [ ] Generate owner tokens with `openssl rand -hex 32` and keep only their digests in config
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

*Last updated: March 2026*

---

## Always-on install (systemd user units)

```bash
make install-units
```

Builds `bin/api` and `bin/leadfeed`; starts MongoDB and Redis through Compose; writes `~/.config/flowra/`
(`config.yaml`, `flowra.env`, 0600) and the user units `flowra-api.service`, `leadfeed-sync.service` and
`leadfeed-sync.timer` (every minute: export, import, deliver answers). The first run generates two tokens,
`owner` (this laptop's browser) and `phone`, and prints them once; the files keep only SHA-256. Repeating the
command refreshes binaries and units and leaves the tokens alone.

Override `LEAD_TASK` (path to `lead-task`), `LEAD_SESSION` and `AGTERM_SOCKET` in the environment on the first
run; they are stored in `flowra.env`.

Remove:

```bash
systemctl --user disable --now flowra-api.service leadfeed-sync.timer
rm ~/.config/systemd/user/{flowra-api.service,leadfeed-sync.service,leadfeed-sync.timer}
rm -r ~/.config/flowra   # also discards the token digests
systemctl --user daemon-reload
```

Units run while the user is logged in; no `sudo` or linger. While the laptop sleeps Flowra is unavailable and
questions wait in the lead's journal.

### What only the owner can do

| Action | Why |
|---|---|
| Save the printed tokens (password manager); enter `phone` in the phone's browser | shown once, needed on the owner's device |
| Set up the phone's SSH client with port forwarding (below) | the phone is the owner's device |
| Confirm that the phone's key in `~/.ssh/authorized_keys` is the owner's | only the owner grants access to the machine |
| Decide on access from outside the home network | a third-party service, `sudo` install, an app on the phone |

### Phone

Access is SSH port forwarding inside the home network: the phone's SSH client forwards a local port to
`127.0.0.1:8080` of the laptop, and the phone's browser opens `http://127.0.0.1:<port>/feed`. The server stays
on loopback, and the token travels inside SSH. The cookie is not `Secure` for this plain-HTTP address; the
form's `Origin` check accepts the forwarded address.

Limits: the phone and the laptop share a network; the laptop is awake; its address may change (DHCP), so fix
it in the client or reserve it in the router. There is no access from outside the home network.
