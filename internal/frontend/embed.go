package frontend

import "embed"

// FS holds the compiled React SPA (frontend/dist built by Vite).
// Dockerfile copies frontend/dist → internal/frontend/dist before go build.
//
// all: prefix includes hidden files like .gitkeep (needed when dist/ only has placeholder)
//
//go:embed all:dist
var FS embed.FS
