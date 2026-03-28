# SkyGuardwall — Agent Instructions

SkyGuardwall 是一个**主机防火墙可视化管理工具**，以 Docker host 网络模式部署，直接读取/写入宿主机 iptables/nftables，通过 Web UI 展示端口暴露情况并管理防火墙规则。

---

## Tech Stack

| Layer | Choice |
|-------|--------|
| Backend | Go 1.22+, Gin, GORM |
| Frontend | React 18, TypeScript, Vite, Ant Design 5.x |
| Auth | golang-jwt/jwt v5, bcrypt |
| Database | GORM + SQLite (default) / MySQL / PostgreSQL |
| Deploy | Docker host network, single port (8080), `go:embed` |

---

## Project Structure

```
sky-guardwall/
├── cmd/server/main.go           # Entry point
├── internal/
│   ├── config/config.go         # Env vars
│   ├── handler/                 # Gin handlers (auth, scan, port, iptables, nft, firewall)
│   ├── middleware/auth.go       # JWT middleware
│   ├── model/                   # GORM models (no FK constraints)
│   ├── service/                 # Business logic
│   │   └── parser/              # ss / iptables / nft output parsers
│   ├── database/db.go           # GORM init + AutoMigrate
│   └── router/router.go         # Route registration
├── frontend/
│   ├── src/
│   │   ├── pages/               # Dashboard, Ports, Iptables, Nft, Firewall
│   │   ├── components/
│   │   ├── api/client.ts        # Axios with Bearer token interceptor
│   │   └── store/auth.ts        # Zustand auth state
│   ├── package.json
│   └── vite.config.ts
├── docs/requirements.md         # Full requirements (read this first)
├── docker-compose.yml
├── Dockerfile
├── go.mod
└── AGENTS.md                    # This file
```

---

## Dev Commands

```bash
# Backend
go mod tidy
go run ./cmd/server          # Start backend (reads env vars)

# Frontend
cd frontend
npm install
npm run dev                  # Dev server with proxy to :8080
npm run build                # Output to frontend/dist (embedded by Go)

# Full build
npm run build --prefix frontend
go build -o sky-guardwall ./cmd/server

# Docker
docker compose up --build
docker compose down
```

---

## Environment Variables

| Var | Default | Notes |
|-----|---------|-------|
| `ADMIN_USER` | `admin` | Admin username |
| `ADMIN_PASS` | `changeme` | Plaintext; bcrypt-hashed at runtime |
| `JWT_SECRET` | random | Set explicitly in production |
| `JWT_EXPIRE_HOURS` | `24` | Token lifetime |
| `DB_TYPE` | `sqlite` | `sqlite` / `mysql` / `postgres` |
| `DB_DSN` | `./data/sky-guardwall.db` | GORM DSN |
| `PORT` | `8080` | Listen port |

---

## Critical Rules (Dos and Don'ts)

### Database
- **NEVER add foreign key constraints** — no `References`, no `constraint:"..."` tags, no `OnDelete/OnUpdate` in GORM. Relationships are managed at the application layer only.
- Use GORM `AutoMigrate` — never write raw DDL migrations.
- Model structs must embed `gorm.Model` or define `ID`, `CreatedAt`, `UpdatedAt` manually.
- All string primary/foreign IDs use `uint` (BIGINT UNSIGNED).

### Backend (Go)
- All handlers live in `internal/handler/`, business logic in `internal/service/`.
- Handlers must not call `os/exec` directly — always go through `internal/service/`.
- System commands (`iptables`, `ss`, `nft`) are executed via `exec.CommandContext` with a timeout (default 10s).
- Return unified JSON: `{"code": 0, "message": "success", "data": ...}` — use a shared `response.go` helper.
- All `/api/v1` routes (except `/auth/login`) go through the JWT middleware in `internal/middleware/auth.go`.
- Never log sensitive data (passwords, tokens).

### Frontend (React/TypeScript)
- All API calls go through `src/api/client.ts` (Axios instance with Bearer token interceptor).
- Pages map 1-to-1 with top-level routes: `/`, `/ports`, `/iptables`, `/nft`, `/firewall`.
- Use Ant Design components exclusively — no mixing with other UI libraries.
- Auth token stored in `localStorage` key `sgw_token`; Zustand store manages auth state.
- On 401 response, redirect to `/login` and clear token.

### Security / System Operations
- All iptables write operations must validate inputs (IP format, port range) before executing.
- Log every iptables write operation (command + result) at INFO level.
- Docker container requires `--privileged` or `CAP_NET_ADMIN` + `CAP_NET_RAW`.
- Never expose raw shell execution to the API — all system commands are whitelisted in service layer.

### Code Style
- Go: standard `gofmt`, no unused imports, error handling required (no `_` for errors in service layer).
- TypeScript: strict mode, named exports, no `any` types.
- Commit messages: `type(scope): description` (e.g., `feat(scan): add UDP port parser`).
- Branch names: `feat/`, `fix/`, `chore/` prefixes.

---

## Architecture: Single Port Deployment

```
Browser → :8080
  ├── /api/v1/*   → Gin API routes (JWT protected)
  └── /*          → Gin serves embedded React SPA (go:embed frontend/dist)
                    React Router handles client-side routing
```

No Nginx needed. Frontend is embedded into the Go binary via `//go:embed frontend/dist`.

---

## Key Design Decisions

1. **No foreign keys** — enables painless DB switching between SQLite/MySQL/PG.
2. **MVP = single admin account** — credentials from env vars, no users table yet.
3. **Read-only nftables** in MVP — write operations only via iptables.
4. **Graceful degradation** — if `nft` not found, return `{"available": false}` rather than 500.
5. **Rule persistence** — DB stores firewall rules; on restart, `is_active=true` rules can be re-applied.
6. **Pure-Go SQLite** (`modernc.org/sqlite`) — avoids CGO for simpler Docker builds; can switch driver if needed.

---

## What NOT to Do

- Do not add dependencies without checking existing `go.mod` / `package.json` first.
- Do not write raw SQL — use GORM methods only.
- Do not add foreign keys or `REFERENCES` constraints — this will break DB compatibility.
- Do not shell out to arbitrary commands — only `iptables`, `iptables-save`, `ss`, `nft`, `ip` are allowed.
- Do not store JWT tokens in the backend DB — stateless JWT only.
- Do not modify `docker-compose.yml` to add new services without asking — single container deployment is a goal.

---

## References

- Full requirements: [`docs/requirements.md`](docs/requirements.md)
- GORM docs: https://gorm.io/docs/
- Gin docs: https://gin-gonic.com/docs/
- Ant Design: https://ant.design/components/overview/
