#!/bin/bash
export PATH=$PATH:/opt/homebrew/bin:/usr/bin:/usr/local/bin:/usr/local/go/bin

cd /Users/jacinlee/selfwork/job/self-proj/sky-guardwall

echo "=== git add ==="
git add -A
echo "ADD=$?"

echo "=== git status ==="
git status --short | head -40

echo "=== git commit ==="
git commit -m "feat: initial MVP implementation

- Go+Gin backend with iptables read/write via exec
- Auto-sync iptables every 60s via background ticker
- GORM models: IptablesRule, ManagedRule, SyncMeta (no FK)
- SQLite default (pure-Go glebarez/sqlite, CGO_ENABLED=0)
- JWT auth via env ADMIN_USER/ADMIN_PASS
- React+Ant Design frontend: iptables viewer + rule manager
- Single port 9176 via go:embed frontend
- Multi-arch Docker image (linux/amd64,linux/arm64)
- GitHub Actions CI/CD → ghcr.io/jacinli/sky-guardwall
- AGENTS.md + CLAUDE.md + .windsurfrules + copilot-instructions
- ACS .agents/ skills structure"
echo "COMMIT=$?"

echo "=== git push ==="
git push origin main
echo "PUSH=$?"
