# SkyGuardwall — GitHub Copilot Instructions

## What This Project Is

Host firewall management tool. Runs as Docker container with host network mode. Reads `iptables`/`nftables`/`ss` output from the host machine. Provides a React web UI on a single port (8080). Go + Gin backend, GORM ORM (SQLite/MySQL/PostgreSQL), JWT auth.

## Stack

- Backend: Go 1.22+, Gin, GORM, golang-jwt/jwt v5, log/slog
- Frontend: React 18, TypeScript (strict), Vite, Ant Design 5.x, Zustand, Axios
- Database: GORM with modernc.org/sqlite (pure-Go, no CGO) / MySQL / PostgreSQL
- Deploy: Docker host network, `go:embed` bundles frontend into Go binary

## Critical Constraints

- **NEVER add foreign keys** — no `gorm:"foreignKey:..."`, no `constraint:`, no `References`. Database must work on SQLite, MySQL, PostgreSQL without changes.
- **System commands whitelist only**: `iptables`, `iptables-save`, `ip6tables`, `ss`, `nft`, `ip`. No shell, no other commands.
- **Validate all iptables write inputs**: IP/CIDR format, port 0–65535, action in ACCEPT/DROP/REJECT only.
- **Handler/Service separation**: handlers do HTTP binding only; services do logic and exec.
- **Frontend API client**: always use `src/api/client.ts` (Axios with Bearer interceptor). Never use `fetch()` directly.
- **Single container**: one Docker service, `network_mode: host`, `privileged: true`. No extra services.

## API Response Format

All endpoints return:
```json
{ "code": 0, "message": "success", "data": { ... } }
```
Errors: `code` = HTTP status, `data` = null.

## File Locations

- `internal/handler/` — Gin handlers (HTTP only)
- `internal/service/` — Business logic + system exec
- `internal/service/parser/` — Pure ss/iptables/nft parsers
- `internal/model/` — GORM structs (no FK tags)
- `internal/database/db.go` — GORM init + AutoMigrate
- `internal/router/router.go` — All routes
- `frontend/src/api/client.ts` — Axios instance
- `frontend/src/store/auth.ts` — Zustand auth
- `frontend/src/pages/` — Dashboard, Ports, Iptables, Nft, Firewall, Login

## Exposure Levels

`public` (0.0.0.0/::), `private` (RFC1918), `loopback` (127.x/::1), `specific` (other)

## Source Types

`docker` (docker-proxy/dockerd/containerd), `system` (sshd/nginx/dnsmasq), `user` (everything else)
