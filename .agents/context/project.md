# SkyGuardwall — Project Context

SkyGuardwall 是一个**主机防火墙可视化管理工具**，以 Docker host 网络模式部署，直接读取/写入宿主机 iptables/nftables，通过 React Web UI 展示端口暴露情况并管理防火墙规则。

## Tech Stack

| Layer    | Choice                                          |
|----------|-------------------------------------------------|
| Backend  | Go 1.22+, Gin, GORM                             |
| Frontend | React 18, TypeScript, Vite, Ant Design 5.x      |
| Auth     | golang-jwt/jwt v5, bcrypt                       |
| Database | GORM + SQLite (default) / MySQL / PostgreSQL    |
| Deploy   | Docker host network, single port 8080, go:embed |

## Project Structure

```
sky-guardwall/
├── cmd/server/main.go           # Entry point; go:embed frontend/dist
├── internal/
│   ├── config/config.go         # All os.Getenv calls here
│   ├── handler/                 # Gin handlers — HTTP binding only, no logic
│   ├── middleware/auth.go       # JWT Bearer middleware
│   ├── model/                   # GORM structs — NO foreign keys
│   ├── service/                 # Business logic + system command execution
│   │   └── parser/              # Pure parsers: ss / iptables / nft output
│   ├── database/db.go           # GORM init, driver switch, AutoMigrate
│   └── router/router.go         # All route registration in one place
├── frontend/
│   ├── src/
│   │   ├── pages/               # Dashboard, Ports, Iptables, Nft, Firewall, Login
│   │   ├── api/client.ts        # Axios instance with Bearer token interceptor
│   │   └── store/auth.ts        # Zustand auth state
│   └── vite.config.ts           # Proxy /api → :8080 in dev
├── docs/requirements.md         # Full requirements — read before coding
├── AGENTS.md                    # Cross-tool agent instructions
├── docker-compose.yml
└── Dockerfile
```

## API Convention

- Base: `/api/v1`
- Auth: `Authorization: Bearer <token>` on all routes except `/api/v1/auth/login`
- Response shape:
  ```json
  { "code": 0, "message": "success", "data": { ... } }
  ```

## Environment Variables

| Var                | Default                    |
|--------------------|----------------------------|
| `ADMIN_USER`       | `admin`                    |
| `ADMIN_PASS`       | `changeme`                 |
| `JWT_SECRET`       | random (warn if missing)   |
| `JWT_EXPIRE_HOURS` | `24`                       |
| `DB_TYPE`          | `sqlite`                   |
| `DB_DSN`           | `./data/sky-guardwall.db`  |
| `PORT`             | `8080`                     |

## Exposure Level Classification

| Value      | Binding Address          | Meaning              |
|------------|--------------------------|----------------------|
| `public`   | `0.0.0.0` / `::`        | All interfaces       |
| `private`  | RFC1918 / ULA            | Private network only |
| `loopback` | `127.0.0.1` / `::1`     | Local only           |
| `specific` | Any other specific IP    | Single interface     |

## Source Type Classification

| Value    | Process Examples                        |
|----------|-----------------------------------------|
| `docker` | docker-proxy, dockerd, containerd       |
| `system` | sshd, nginx, dnsmasq, systemd, dnsmasq  |
| `user`   | everything else                         |

## Key Design Decisions

1. **No FK constraints** — works with SQLite / MySQL / PostgreSQL without changes.
2. **Single admin account (MVP)** — credentials from env vars, no users table.
3. **nftables read-only (MVP)** — write only via iptables.
4. **Graceful degradation** — `nft` not found → `{"available": false}`, not 500.
5. **Pure-Go SQLite** (`modernc.org/sqlite`) — CGO_ENABLED=0, simpler Docker build.
6. **Single port** — Go binary embeds React SPA via `go:embed`; no Nginx needed.
