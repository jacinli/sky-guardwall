# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci --prefer-offline
COPY frontend/ .
# Build output goes to ../internal/frontend/dist (configured in vite.config.ts)
RUN npm run build

# Stage 2: Build Go binary (CGO_ENABLED=0 — uses pure-Go sqlite via glebarez/sqlite)
FROM golang:1.25-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
# Vite builds to internal/frontend/dist (configured in vite.config.ts outDir)
COPY --from=frontend-builder /app/internal/frontend/dist ./internal/frontend/dist
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o sky-guardwall ./cmd/server

# Stage 3: Minimal runtime image
FROM alpine:3.19
# iptables, ip6tables: firewall tools; iproute2: provides ss; nftables: nft command
RUN apk add --no-cache iptables ip6tables iproute2 nftables ca-certificates tzdata
WORKDIR /app
COPY --from=backend-builder /app/sky-guardwall .
VOLUME ["/data"]
EXPOSE 9176
ENTRYPOINT ["/app/sky-guardwall"]
