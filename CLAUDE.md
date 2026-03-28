# SkyGuardwall — Claude Code Instructions

@docs/requirements.md
@AGENTS.md

---

## Project in One Sentence

Docker host-network firewall manager: reads `iptables`/`nftables`/`ss` from the host, shows port exposure in a React UI, lets you add/delete iptables rules — Go + Gin backend, single port, go:embed frontend.

## Commands

```bash
# Backend dev
go mod tidy && go run ./cmd/server

# Frontend dev (proxies /api → :8080)
cd frontend && npm install && npm run dev

# Full build (frontend must build first — Go embeds the dist/)
npm run build --prefix frontend
go build -o sky-guardwall ./cmd/server

# Docker
docker compose up --build
docker compose logs -f
```

## The Rules Claude Must Never Break

1. **No database foreign keys** — `gorm:"foreignKey:..."`, `References:`, `constraint:` are all forbidden. The DB must work on SQLite / MySQL / PostgreSQL without schema changes.

2. **System commands are whitelisted** — only `iptables`, `iptables-save`, `ip6tables`, `ss`, `nft`, `ip` may be called via `exec.CommandContext`. No `bash -c` or shell with user input.

3. **All inputs to iptables writes must be validated** — IP/CIDR format, port 0–65535, action in `{ACCEPT,DROP,REJECT}`.

4. **Handler → Service separation** — handlers bind HTTP params and call services; services execute logic and commands. Never exec in handlers.

5. **Frontend API calls through `src/api/client.ts` only** — never use raw `fetch()` or create ad-hoc Axios instances.

6. **Single container, host network** — `docker-compose.yml` has one service with `network_mode: host` and `privileged: true`. Do not add services.

## Architecture Reminder

```
Browser
  │
  ├── /api/v1/*  →  Gin (JWT middleware)  →  Handler  →  Service  →  exec / GORM
  └── /*         →  Gin serves go:embed React SPA
```

## Skill Hints

For detailed conventions, reference `.agents/skills/`:
- `go-backend/` — handler patterns, JWT, response helper, slog
- `react-frontend/` — Axios client, Zustand, Ant Design color mapping
- `database/` — GORM model definitions, no-FK query patterns
- `system-ops/` — command whitelist, input validation, ss parser, exposure classifier
- `docker-deploy/` — Dockerfile, compose, go:embed requirement
