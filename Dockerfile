# syntax=docker/dockerfile:1

# ============================================================================
# gosearx — multi-stage build.
#
# gosearx is a CGO-free static Go binary with the frontend (web/dist) and the
# bangs DB embedded via go:embed. Only engines/, plugins/, and settings.yml are
# read from disk at runtime.
#
# The runtime is debian-slim with bash + python3 so EVERY plugin backend works:
# native Go, in-process Lua (.lua) and JS (.mjs), and exec/script plugins
# (.sh/.py/.rb/.pl).
#
#   docker build -t gosearx .
#   docker run --rm -p 8080:8080 gosearx
#
# Valkey/Redis runs externally; set settings.yml's `valkey.url`
# (e.g. valkey://host:6379/1) or leave it empty for the in-memory fallback.
# ============================================================================

# ---- Stage 1: build the frontend (embedded by the Go build) ----
FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci || npm install
COPY web/ ./
RUN npm run build

# ---- Stage 2: compile the static Go binary ----
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Provide the freshly built frontend so `go:embed all:dist` has content.
COPY --from=web /src/web/dist ./web/dist
# Fully static, stripped binary (pure Go, no CGO).
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -trimpath -ldflags="-s -w" -o /out/gosearx ./cmd/gosearx

# ---- Stage 3: runtime (debian-slim + bash + python3 for exec plugins) ----
FROM debian:stable-slim AS runtime
LABEL org.opencontainers.image.source="https://github.com/abs3ntdev/gosearx" \
      org.opencontainers.image.description="gosearx — a Go metasearch engine with Lua/JS/exec engines & plugins, AI answer synthesis, and interactive finance charts." \
      org.opencontainers.image.licenses="AGPL-3.0"
WORKDIR /app
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates bash python3 curl \
 && rm -rf /var/lib/apt/lists/* \
 && useradd -u 65532 -m -s /usr/sbin/nologin gosearx
# Binary + disk-loaded data (frontend & bangs are already embedded).
COPY --from=build /out/gosearx /app/gosearx
COPY engines/ /app/engines/
COPY plugins/ /app/plugins/
COPY settings.yml /app/settings.yml
RUN chown -R 65532:65532 /app
# Built-in plugins live in /app/plugins. Mount EXTRA/custom plugins at
# /app/custom-plugins (they add to, never hide, the built-ins).
ENV GOSEARX_PLUGINS=/app/custom-plugins
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/app/gosearx"]
CMD ["serve", "-addr", ":8080"]
