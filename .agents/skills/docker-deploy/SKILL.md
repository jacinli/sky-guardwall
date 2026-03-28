---
name: docker-deploy
description: Use when writing or editing Dockerfile, docker-compose.yml, .dockerignore, or anything related to building and deploying the container. Enforces host network mode and single-container constraints.
---

# Docker Deployment Conventions

## Non-Negotiable Constraints

1. `network_mode: host` — container shares host network stack; no `ports:` mapping
2. `privileged: true` — required for iptables execution inside container
3. Single container — no sidecars, no nginx proxy, no separate frontend service
4. Single port `:8080` — Go binary serves both API and embedded React SPA
5. `CGO_ENABLED=0` — use `modernc.org/sqlite` (pure-Go); no gcc needed

## docker-compose.yml

```yaml
version: "3.9"

services:
  sky-guardwall:
    image: sky-guardwall:latest
    build:
      context: .
      dockerfile: Dockerfile
    network_mode: host         # shares host iptables/ss
    privileged: true           # iptables write access
    restart: unless-stopped
    environment:
      - ADMIN_USER=admin
      - ADMIN_PASS=changeme    # change in production
      - JWT_SECRET=replace-with-32char-random-secret
      - JWT_EXPIRE_HOURS=24
      - DB_TYPE=sqlite
      - DB_DSN=/data/sky-guardwall.db
      - PORT=8080
    volumes:
      - ./data:/data           # persist SQLite database
```

**Do NOT add `ports:` mapping with host network mode.**  
**Do NOT add more services.**

## Dockerfile (Multi-Stage)

```dockerfile
# Stage 1 — build frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci --prefer-offline
COPY frontend/ .
RUN npm run build

# Stage 2 — build Go binary (no CGO)
FROM golang:1.22-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o sky-guardwall ./cmd/server

# Stage 3 — runtime
FROM alpine:3.19
RUN apk add --no-cache iptables ip6tables nftables iproute2
WORKDIR /app
COPY --from=backend-builder /app/sky-guardwall .
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["/app/sky-guardwall"]
```

Required runtime packages: `iptables`, `ip6tables`, `nftables`, `iproute2` (provides `ss`).

## .dockerignore

```
.git
.agents
frontend/node_modules
frontend/dist
data/
*.md
```

## go:embed Requirement

Must exist in `cmd/server/main.go` (or a `_embed.go` file in same package):

```go
//go:embed frontend/dist
var frontendFS embed.FS
```

`frontend/dist` must exist at `go build` time — the Dockerfile copies it before building Go.

## Startup Behavior

- If `JWT_SECRET` is empty: log a WARNING and generate a random secret (tokens invalidated on restart)
- Log `ADMIN_PASS` as `[REDACTED]`
- `DB_DSN` for SQLite should default to `/data/sky-guardwall.db`

## Health Endpoint (no auth)

```
GET /api/v1/health → {"code":0,"message":"ok","data":null}
```
