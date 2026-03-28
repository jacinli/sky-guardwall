---
name: go-backend
description: Use when writing or editing Go backend code — Gin handlers, GORM models, service layer, JWT middleware, system command execution, or unified response helpers. Do NOT trigger for frontend TypeScript/React files or Dockerfile.
---

# Go Backend Conventions

## Layer Separation (strict)

- `internal/handler/` — HTTP binding, param validation, call service, return response. Zero business logic.
- `internal/service/` — All business logic. Only place allowed to call `exec.CommandContext`.
- `internal/service/parser/` — Pure parsing functions, no side effects, no I/O.
- `internal/model/` — GORM struct definitions only.
- `internal/router/router.go` — All route registration in one file.

## Handler Pattern

```go
func (h *ScanHandler) RunScan(c *gin.Context) {
    result, err := h.scanService.Run(c.Request.Context())
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    response.Success(c, result)
}
```

**Never call `c.JSON` directly in handlers** — always use `response.Success` / `response.Error`.

## Unified Response Helper

```go
// internal/response/response.go
type R struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data"`
}
func Success(c *gin.Context, data any) {
    c.JSON(http.StatusOK, R{Code: 0, Message: "success", Data: data})
}
func Error(c *gin.Context, status int, msg string) {
    c.JSON(status, R{Code: status, Message: msg, Data: nil})
}
```

## System Command Execution

Only these commands may be called via `exec.CommandContext`:
`iptables`, `iptables-save`, `ip6tables`, `ss`, `nft`, `ip`

```go
func execCmd(ctx context.Context, name string, args ...string) (string, error) {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    var stdout, stderr bytes.Buffer
    cmd := exec.CommandContext(ctx, name, args...)
    cmd.Stdout, cmd.Stderr = &stdout, &stderr
    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("%s %v: %w (stderr: %s)", name, args, err, stderr.String())
    }
    return stdout.String(), nil
}
```

**Never pass user input directly into args. Always construct args explicitly.**

## JWT Middleware

```go
// internal/middleware/auth.go
func JWTAuth(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        raw := c.GetHeader("Authorization")
        tokenStr := strings.TrimPrefix(raw, "Bearer ")
        // parse + validate, set c.Set("username", claims.Subject)
        c.Next()
    }
}
```

Apply to all `/api/v1` routes except `/api/v1/auth/login`.

## GORM Model Rules

- **No foreign key tags**: no `gorm:"foreignKey:..."`, no `References`, no `constraint`
- Use `uint` for all ID fields
- Long text: `gorm:"type:text"`
- Use `time.Time` not pointer for required timestamps

## Error Handling

- Never discard errors with `_` in service or model layer
- Wrap with context: `fmt.Errorf("scan run: %w", err)`
- HTTP 400 bad input / 401 auth / 500 internal

## Logging

Use `log/slog` (Go stdlib 1.21+). Never log passwords or tokens.

```go
slog.Info("iptables write", "rule_id", id, "args", args)
slog.Error("exec failed", "err", err)
```

## go:embed

```go
//go:embed frontend/dist
var frontendFS embed.FS
// Router: serve SPA for all non-/api routes
```
