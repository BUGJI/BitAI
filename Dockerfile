# BitAPI - Multi-stage Docker build
# Stage 1: Build Vue 3 frontend
FROM node:20-alpine AS frontend-build
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go backend (CGO_ENABLED=0, glebarez/sqlite is pure Go)
FROM golang:1.23-alpine AS backend-build
ENV GOTOOLCHAIN=auto
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server

# Stage 3: Minimal runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates curl
WORKDIR /app
COPY --from=frontend-build /app/frontend/dist /app/frontend/dist
COPY --from=backend-build /app/server /app/server

RUN mkdir -p /app/data/uploads/avatars

EXPOSE 8091

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -fsS http://localhost:8091/health || exit 1

ENV BITAPI_DATABASE_DSN="file:/app/data/bitapi.db?_foreign_keys=on&_busy_timeout=5000"

ENTRYPOINT ["/app/server"]