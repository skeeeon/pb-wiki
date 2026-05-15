# syntax=docker/dockerfile:1.7

# -----------------------------------------------------------------------------
# Stage 1 — frontend
# -----------------------------------------------------------------------------
FROM node:22-alpine AS frontend
WORKDIR /app/frontend

# Install deps first so the layer caches when only source changes.
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build
# After this stage: /app/frontend/dist contains index.html + hashed assets.

# -----------------------------------------------------------------------------
# Stage 2 — backend (embeds the frontend dist via //go:embed)
# -----------------------------------------------------------------------------
FROM golang:1.25-alpine AS backend
WORKDIR /app

# Module cache first.
COPY go.mod go.sum ./
RUN go mod download

# Source. Copy frontend/dist from the previous stage so //go:embed has files
# to bundle.
COPY main.go frontend.go ./
COPY internal/ ./internal/
COPY migrations/ ./migrations/
COPY --from=frontend /app/frontend/dist ./frontend/dist

# CGO off → fully static binary. Trim symbol tables for size.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/pb-wiki .

# -----------------------------------------------------------------------------
# Stage 3 — runtime
# -----------------------------------------------------------------------------
FROM alpine:3
RUN apk add --no-cache ca-certificates tini \
    && adduser -D -u 1000 pbwiki
USER pbwiki
WORKDIR /home/pbwiki

COPY --from=backend /out/pb-wiki /usr/local/bin/pb-wiki

EXPOSE 8090
VOLUME ["/home/pbwiki/pb_data"]

# tini reaps zombies and forwards signals so Ctrl-C cleanly stops PocketBase.
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/pb-wiki"]
CMD ["serve", "--http", "0.0.0.0:8090"]
