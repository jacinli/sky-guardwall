# Stage 1: Build frontend (always on build platform — output is just files, no arch dependency)
FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci --prefer-offline
COPY frontend/ .
# Vite outDir is ../internal/frontend/dist (see vite.config.ts)
RUN npm run build

# Stage 2: Cross-compile Go binary on the BUILD platform (native amd64 runner)
# Using --platform=$BUILDPLATFORM means this stage always runs natively on the CI host,
# never via QEMU. GOOS/GOARCH env vars tell Go to cross-compile for the target arch.
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS backend-builder
ARG TARGETOS=linux
ARG TARGETARCH=amd64
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
# Copy compiled frontend assets into the embed package location
COPY --from=frontend-builder /app/internal/frontend/dist ./internal/frontend/dist
COPY . .
# CGO_ENABLED=0 + GOOS/GOARCH = true cross-compilation, no QEMU needed
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o sky-guardwall ./cmd/server

# Stage 3: Minimal runtime image (this stage runs on the target platform — tiny, no compilation)
FROM alpine:3.19
# iptables/ip6tables: firewall rules; iproute2: provides `ss`; nftables: nft command
RUN apk add --no-cache iptables ip6tables iproute2 nftables ca-certificates tzdata
WORKDIR /app
COPY --from=backend-builder /app/sky-guardwall .
VOLUME ["/data"]
EXPOSE 9176
ENTRYPOINT ["/app/sky-guardwall"]
