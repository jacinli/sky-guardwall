# SkyGuardwall 需求文档

> 版本：v0.1（MVP）  
> 最后更新：2026-03-28

---

## 目录

1. [项目背景与目标](#1-项目背景与目标)
2. [系统架构](#2-系统架构)
3. [核心功能模块](#3-核心功能模块)
4. [数据模型](#4-数据模型)
5. [API 设计](#5-api-设计)
6. [技术选型](#6-技术选型)
7. [部署方案](#7-部署方案)
8. [目录结构规划](#8-目录结构规划)
9. [待确认事项](#9-待确认事项)

---

## 1. 项目背景与目标

### 1.1 背景

服务器在公网运行时，各类端口（系统进程、Docker 容器、用户服务）会以不同形式暴露。现有工具（如 `ss`、`iptables`）需要通过命令行逐条排查，缺少统一的可视化视图与集中管理能力。SkyGuardwall 旨在填补这一空缺。

### 1.2 目标

- 以 Docker host 网络模式部署，直接读取宿主机网络栈信息
- 扫描整台机器（不只是 Docker 容器）所有监听端口，识别公网/私网暴露情况
- 区分端口来源（Docker daemon、docker-proxy、系统进程、用户程序等）
- 解析并可视化展示现有 iptables / nftables 规则
- 提供防火墙规则的增删管理能力，规则持久化到数据库
- 前后端单端口部署，通过账号密码 + JWT 认证保护所有操作

### 1.3 MVP 范围

| 功能 | 是否 MVP |
|------|---------|
| 端口扫描与暴露分析 | ✅ |
| iptables 规则读取展示 | ✅ |
| nftables 规则读取展示 | ✅ |
| 防火墙规则写入（添加/删除） | ✅ |
| JWT 认证 | ✅ |
| 扫描历史记录 | ✅ |
| 规则导入/导出 | ❌（后续版本） |
| 多用户管理 | ❌（后续版本） |
| 告警/通知推送 | ❌（后续版本） |

---

## 2. 系统架构

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                     Docker 容器（host 网络）              │
│                                                         │
│  ┌──────────────┐     ┌──────────────────────────────┐  │
│  │  React SPA   │     │        Gin HTTP Server        │  │
│  │  (embedded   │◄────│        :8080 (单端口)         │  │
│  │   go:embed)  │     │                              │  │
│  └──────────────┘     │  ┌─────────┐  ┌──────────┐  │  │
│                        │  │ 扫描引擎 │  │ 规则引擎  │  │  │
│                        │  └────┬────┘  └────┬─────┘  │  │
│                        │       │             │        │  │
│                        │  ┌────▼─────────────▼─────┐ │  │
│                        │  │        ORM (GORM)       │ │  │
│                        │  └────────────┬────────────┘ │  │
│                        └───────────────┼──────────────┘  │
│                                        │                  │
│  宿主机系统调用                          ▼                  │
│  ┌────────────────────┐    ┌───────────────────────┐     │
│  │ iptables / nft     │    │  SQLite / MySQL / PG   │     │
│  │ ss -ltnp / -lunp   │    └───────────────────────┘     │
│  └────────────────────┘                                   │
└─────────────────────────────────────────────────────────┘
```

### 2.2 请求流程

```
浏览器
  │
  ├─ GET /* ────────────────────────► Gin 静态文件服务（embed React build）
  │
  └─ /api/v1/* ─► JWT 鉴权中间件 ─► Controller ─► Service ─► 执行系统命令 / GORM
```

### 2.3 单端口原则

- 前端由 Vite 构建成静态产物，通过 `go:embed` 打包进 Go binary
- Gin 优先匹配 `/api/v1/*` 路由；其余所有路径返回 `index.html`，由 React Router 处理客户端路由
- 无需 Nginx，无需独立前端服务

---

## 3. 核心功能模块

### 3.1 端口扫描与暴露分析

#### 采集命令

| 命令 | 用途 |
|------|------|
| `ss -ltnp` | 获取 TCP 监听端口及进程信息 |
| `ss -lunp` | 获取 UDP 监听端口及进程信息 |

#### 解析字段

从 `ss` 输出中提取：

- `Local Address:Port`：绑定的本地地址和端口
- `Process`：进程名称与 PID（`users:(("docker-proxy",pid=3631914,fd=7))`）
- 协议类型：TCP / UDP

#### 暴露等级判定

| 绑定地址 | 暴露等级 | 说明 |
|---------|---------|------|
| `0.0.0.0:PORT` 或 `[::]:PORT` | **public** | 所有网卡，含公网接口 |
| `10.x.x.x` / `172.16-31.x.x` / `192.168.x.x` | **private** | 仅私网可达 |
| `127.0.0.1` / `[::1]` | **loopback** | 仅本机 |
| 其他具体 IP | **specific** | 绑定到特定网卡 |

#### 来源类型区分

| 进程名关键字 | 来源类型 |
|------------|---------|
| `docker-proxy` | docker |
| `dockerd` | docker |
| `containerd` | docker |
| `sshd` / `nginx` / `dnsmasq` / `systemd` 等系统进程 | system |
| 其他 | user |

### 3.2 iptables 规则读取

- 执行 `iptables -S` 获取所有规则的原始文本
- 解析结构：策略行（`-P CHAIN ACTION`）、规则行（`-A CHAIN ...`）
- 展示维度：
  - 按链分组（INPUT / FORWARD / OUTPUT / 自定义链）
  - 来自 fail2ban 的 `f2b-*` 链单独标记
  - 来自 Docker 的 `DOCKER*` 链单独标记
- 同步读取 `iptables-save` 获取完整规则集（含 table 信息：filter / nat / mangle / raw）

### 3.3 nftables 规则读取

- 执行 `nft list ruleset` 获取完整规则集原始文本
- 解析 table、chain、rule 层级结构
- 若命令不存在（系统未安装 nft），graceful 降级，返回空结果

### 3.4 防火墙规则管理（写操作）

#### 支持的操作

| 操作 | iptables 命令示例 |
|------|-----------------|
| 封禁来源 IP | `iptables -I INPUT -s <IP> -j DROP` |
| 封禁目标端口 | `iptables -I INPUT -p tcp --dport <PORT> -j DROP` |
| 允许来源 IP | `iptables -I INPUT -s <IP> -j ACCEPT` |
| 白名单端口+来源 IP | `iptables -I INPUT -s <IP> -p tcp --dport <PORT> -j ACCEPT` |
| 删除规则 | `iptables -D INPUT <行号>` 或按条件匹配删除 |

#### 规则持久化

- 每次通过界面添加/删除的规则，记录到数据库 `firewall_rules` 表
- 应用状态字段 `is_active` 标记规则是否已写入 iptables
- 服务重启后可选择"重新应用所有 active 规则"（按创建时间顺序）

### 3.5 认证与授权

- MVP 仅支持单管理员账号
- 账号密码通过环境变量注入（`ADMIN_USER` / `ADMIN_PASS`）
- 密码在内存中以 bcrypt 比对（不存数据库，直接对比 env 明文）
- 登录成功返回 JWT（有效期 24 小时，可配置）
- 前端将 token 存入 `localStorage`，每次请求携带 `Authorization: Bearer <token>`
- Gin 中间件统一鉴权，鉴权失败返回 401

---

## 4. 数据模型

> 所有表不使用外键约束，关联关系通过应用层维护。  
> ORM 使用 GORM，支持 SQLite / MySQL / PostgreSQL。

### 4.1 scan_records 表

```
scan_records
├── id            BIGINT UNSIGNED  PK AUTO_INCREMENT
├── scanned_at    DATETIME         扫描触发时间
├── raw_ss_tcp    TEXT             ss -ltnp 原始输出
├── raw_ss_udp    TEXT             ss -lunp 原始输出
├── raw_iptables  TEXT             iptables -S 原始输出
├── raw_nft       TEXT             nft list ruleset 原始输出（可为空）
└── created_at    DATETIME
```

### 4.2 port_entries 表

```
port_entries
├── id              BIGINT UNSIGNED  PK AUTO_INCREMENT
├── scan_record_id  BIGINT UNSIGNED  关联 scan_records.id（无 FK 约束）
├── protocol        VARCHAR(8)       tcp / udp
├── local_addr      VARCHAR(64)      绑定地址，如 0.0.0.0 / 192.168.1.1
├── port            INT              端口号
├── process_name    VARCHAR(128)     进程名
├── pid             INT              进程 PID
├── exposure_level  VARCHAR(16)      public / private / loopback / specific
├── source_type     VARCHAR(16)      docker / system / user
└── created_at      DATETIME
```

### 4.3 firewall_rules 表

```
firewall_rules
├── id          BIGINT UNSIGNED  PK AUTO_INCREMENT
├── rule_type   VARCHAR(16)      allow / deny
├── direction   VARCHAR(8)       in / out
├── protocol    VARCHAR(8)       tcp / udp / all
├── src_ip      VARCHAR(64)      来源 IP 或 CIDR（空表示 any）
├── dst_port    INT              目标端口（0 表示 any）
├── action      VARCHAR(16)      ACCEPT / DROP / REJECT
├── chain       VARCHAR(32)      iptables 链名，默认 INPUT
├── comment     VARCHAR(256)     备注说明
├── is_active   BOOLEAN          是否已应用到 iptables
├── created_at  DATETIME
└── updated_at  DATETIME
```

> **注**：`users` 表在 MVP 中不存储用户信息，认证凭证完全来自环境变量。若后续版本需要多用户，再添加 users 表。

---

## 5. API 设计

### 基础约定

- Base URL：`/api/v1`
- 请求/响应格式：`application/json`
- 认证：除登录接口外，所有接口需携带 `Authorization: Bearer <token>`
- 统一响应结构：

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

错误时 `code` 非零，`message` 描述错误原因，`data` 为 null。

### 5.1 认证接口

#### POST /api/v1/auth/login

登录获取 JWT。

**Request Body：**

```json
{
  "username": "admin",
  "password": "your_password"
}
```

**Response：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2026-03-29T12:00:00Z"
  }
}
```

### 5.2 扫描接口

#### POST /api/v1/scan/run

触发一次完整扫描（ss + iptables + nft），解析结果存库，返回扫描 ID 和解析结果。

**Response data：**

```json
{
  "scan_id": 42,
  "scanned_at": "2026-03-28T10:00:00Z",
  "port_entries": [
    {
      "protocol": "tcp",
      "local_addr": "0.0.0.0",
      "port": 8080,
      "process_name": "docker-proxy",
      "pid": 1148773,
      "exposure_level": "public",
      "source_type": "docker"
    }
  ],
  "summary": {
    "total": 25,
    "public": 8,
    "private": 12,
    "loopback": 5
  }
}
```

#### GET /api/v1/scan/history

获取历史扫描列表（分页）。

**Query Params：** `page`（默认 1）、`page_size`（默认 20）

#### GET /api/v1/scan/history/:id

获取某次扫描的完整详情，含原始命令输出和解析结果。

### 5.3 端口接口

#### GET /api/v1/ports

返回最近一次扫描的端口列表，支持过滤。

**Query Params：**

| 参数 | 说明 |
|------|------|
| `exposure_level` | public / private / loopback / specific |
| `source_type` | docker / system / user |
| `protocol` | tcp / udp |
| `scan_id` | 指定某次扫描，默认取最近一次 |

### 5.4 iptables 接口

#### GET /api/v1/iptables/rules

实时执行 `iptables -S`，返回解析后的规则列表，按链分组。

**Response data：**

```json
{
  "raw": "-P INPUT ACCEPT\n-A INPUT ...",
  "chains": {
    "INPUT": [
      { "action": "ACCEPT", "src": "192.168.1.0/24", "protocol": "tcp", "dport": 22 }
    ],
    "DOCKER": [ ... ],
    "f2b-sshd": [ ... ]
  }
}
```

### 5.5 nftables 接口

#### GET /api/v1/nft/rules

实时执行 `nft list ruleset`，返回原始文本和基本解析结构。

```json
{
  "available": true,
  "raw": "table ip raw { ... }",
  "tables": [
    {
      "name": "ip raw",
      "chains": ["PREROUTING"]
    }
  ]
}
```

### 5.6 防火墙规则管理接口

#### GET /api/v1/firewall/rules

列出数据库中所有用户管理的规则。

**Query Params：** `is_active`（true/false）、`rule_type`（allow/deny）

#### POST /api/v1/firewall/rules

添加并立即应用一条防火墙规则。

**Request Body：**

```json
{
  "rule_type": "deny",
  "direction": "in",
  "protocol": "tcp",
  "src_ip": "216.180.127.201",
  "dst_port": 0,
  "action": "DROP",
  "chain": "INPUT",
  "comment": "手动封禁扫描 IP"
}
```

#### DELETE /api/v1/firewall/rules/:id

从 iptables 删除规则，并将数据库记录的 `is_active` 置为 false。

#### POST /api/v1/firewall/rules/:id/toggle

切换规则启用/禁用状态（对应 iptables 的插入/删除）。

---

## 6. 技术选型

### 6.1 后端

| 组件 | 选型 | 说明 |
|------|------|------|
| 语言 | Go 1.22+ | - |
| HTTP 框架 | Gin | 路由、中间件、参数绑定 |
| ORM | GORM | 支持 SQLite / MySQL / PostgreSQL |
| JWT | `golang-jwt/jwt/v5` | 生成和验证 token |
| 密码哈希 | `golang.org/x/crypto/bcrypt` | 对比 env 密码（内存级别） |
| 配置管理 | 环境变量 + `os.Getenv` | 不引入额外配置库 |
| 静态文件 | `embed.FS` + `net/http` | 内嵌 React 构建产物 |

### 6.2 前端

| 组件 | 选型 | 说明 |
|------|------|------|
| 框架 | React 18 + TypeScript | - |
| 构建工具 | Vite | 快速构建，输出 dist/ |
| UI 组件库 | Ant Design 5.x | 表格、表单、标签等控件丰富 |
| 状态管理 | Zustand | 轻量，适合 MVP |
| HTTP 客户端 | Axios | 统一 interceptor 处理 token |
| 路由 | React Router v6 | 客户端路由 |

### 6.3 数据库

| DB | GORM Driver | 适用场景 |
|----|------------|---------|
| SQLite | `gorm.io/driver/sqlite` | 默认，单机部署，零依赖 |
| MySQL | `gorm.io/driver/mysql` | 有现成 MySQL 实例时 |
| PostgreSQL | `gorm.io/driver/postgres` | 有现成 PG 实例时 |

通过环境变量 `DB_TYPE`（sqlite / mysql / postgres）和 `DB_DSN` 切换，GORM AutoMigrate 自动建表。

---

## 7. 部署方案

### 7.1 环境变量

| 变量 | 说明 | 默认值 |
|------|------|-------|
| `ADMIN_USER` | 管理员用户名 | `admin` |
| `ADMIN_PASS` | 管理员密码（明文，启动时 bcrypt hash） | `changeme` |
| `JWT_SECRET` | JWT 签名密钥 | 随机生成（每次重启失效） |
| `JWT_EXPIRE_HOURS` | Token 有效期（小时） | `24` |
| `DB_TYPE` | 数据库类型：sqlite / mysql / postgres | `sqlite` |
| `DB_DSN` | 数据库连接字符串 | `./data/sky-guardwall.db` |
| `PORT` | 服务监听端口 | `8080` |

### 7.2 docker-compose.yml 示例

```yaml
version: "3.9"

services:
  sky-guardwall:
    image: sky-guardwall:latest
    build:
      context: .
      dockerfile: Dockerfile
    network_mode: host
    privileged: true          # 执行 iptables 需要特权
    restart: unless-stopped
    environment:
      - ADMIN_USER=admin
      - ADMIN_PASS=changeme   # 生产环境请修改
      - JWT_SECRET=your-secret-key-change-this
      - DB_TYPE=sqlite
      - DB_DSN=/data/sky-guardwall.db
      - PORT=8080
    volumes:
      - ./data:/data          # 持久化 SQLite 数据库
```

### 7.3 Dockerfile 思路

```dockerfile
# 多阶段构建
# Stage 1: 构建前端
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# Stage 2: 构建后端（含 embed 前端产物）
FROM golang:1.22-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o sky-guardwall ./cmd/server

# Stage 3: 运行镜像
FROM alpine:3.19
RUN apk add --no-cache iptables nftables iproute2
COPY --from=backend-builder /app/sky-guardwall /usr/local/bin/
ENTRYPOINT ["sky-guardwall"]
```

> SQLite 需要 CGO，故基础镜像用 `alpine` + `gcc`，或改用 pure-go SQLite 驱动（`modernc.org/sqlite`）规避 CGO 依赖。

---

## 8. 目录结构规划

```
sky-guardwall/
├── cmd/
│   └── server/
│       └── main.go              # 程序入口
├── internal/
│   ├── config/
│   │   └── config.go            # 环境变量读取与初始化
│   ├── handler/
│   │   ├── auth.go              # 认证相关 handler
│   │   ├── scan.go              # 扫描相关 handler
│   │   ├── port.go              # 端口列表 handler
│   │   ├── iptables.go          # iptables 读取 handler
│   │   ├── nft.go               # nftables 读取 handler
│   │   └── firewall.go          # 防火墙规则管理 handler
│   ├── middleware/
│   │   └── auth.go              # JWT 鉴权中间件
│   ├── model/
│   │   ├── scan_record.go       # scan_records 表模型
│   │   ├── port_entry.go        # port_entries 表模型
│   │   └── firewall_rule.go     # firewall_rules 表模型
│   ├── service/
│   │   ├── scan.go              # 扫描业务逻辑
│   │   ├── parser/
│   │   │   ├── ss.go            # ss 输出解析
│   │   │   ├── iptables.go      # iptables -S 输出解析
│   │   │   └── nft.go           # nft list ruleset 输出解析
│   │   ├── iptables.go          # iptables 读/写操作封装
│   │   └── firewall.go          # 防火墙规则业务逻辑
│   ├── database/
│   │   └── db.go                # GORM 初始化与 AutoMigrate
│   └── router/
│       └── router.go            # Gin 路由注册
├── frontend/
│   ├── src/
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx    # 首页：扫描概览
│   │   │   ├── Ports.tsx        # 暴露端口列表
│   │   │   ├── Iptables.tsx     # iptables 规则展示
│   │   │   ├── Nft.tsx          # nftables 规则展示
│   │   │   └── Firewall.tsx     # 防火墙规则管理
│   │   ├── components/
│   │   ├── api/
│   │   │   └── client.ts        # Axios 封装
│   │   └── store/
│   │       └── auth.ts          # Zustand 认证状态
│   ├── package.json
│   └── vite.config.ts
├── docs/
│   └── requirements.md          # 本文档
├── data/                        # SQLite 数据持久化目录（gitignore）
├── docker-compose.yml
├── Dockerfile
├── go.mod
└── go.sum
```

---

## 9. 待确认事项

以下问题需要在开发前进一步明确：

1. **防火墙写操作是否需要"预览 → 确认"两步交互？**  
   还是直接点击"添加规则"即立即写入 iptables？

2. **规则持久化重启恢复策略？**  
   服务重启后是否自动将数据库中 `is_active=true` 的规则重新写入 iptables？（iptables 规则在系统重启后默认会丢失，除非有 iptables-persistent 等工具）

3. **nftables 写操作？**  
   MVP 是否只读展示 nftables，还是也需要通过 `nft` 命令添加/删除规则？

4. **iptables 与 nftables 并存时的处理策略？**  
   现代 Linux 系统上 `iptables` 可能是 `nft` 的前端（iptables-nft），写操作只走 iptables 即可，还是需要区分？

5. **前端 UI 风格偏好？**  
   已选 Ant Design，是否认可？还是偏好其他（如 shadcn/ui + Tailwind 的更现代风格）？

6. **扫描触发方式？**  
   除了手动点击"立即扫描"，是否需要定时自动扫描（如每 5 分钟），并在端口变化时高亮提示？

7. **IP 黑名单批量导入？**  
   是否需要支持粘贴一批 IP/CIDR，批量生成封禁规则？

---

*本文档为 MVP 初稿，待上述问题确认后更新。*
