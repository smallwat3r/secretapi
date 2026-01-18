# frontend build
FROM node:20-alpine AS frontend-builder

WORKDIR /src/web
COPY web/package.json web/package-lock.json* ./
RUN --mount=type=cache,id=${RAILWAY_CACHE_KEY:-local}-npm,target=/root/.npm \
    npm ci --prefer-offline
COPY web .
RUN npm run build

# backend build
FROM golang:1.24-alpine AS builder

ENV CGO_ENABLED=0 GOOS=linux GO111MODULE=on

WORKDIR /src

# Copy and download dependencies first (better layer caching)
COPY go.mod go.sum ./
RUN --mount=type=cache,id=${RAILWAY_CACHE_KEY:-local}-go-mod,target=/go/pkg/mod \
    go mod download

# Copy source code (excluding frontend which is copied separately)
COPY cmd ./cmd
COPY internal ./internal

# Copy frontend build output
COPY --from=frontend-builder /src/web/static/dist ./web/static/dist

# Copy web assets needed at runtime
COPY web/robots.txt ./web/robots.txt

# Build with cache mount for faster rebuilds
RUN --mount=type=cache,id=${RAILWAY_CACHE_KEY:-local}-go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=${RAILWAY_CACHE_KEY:-local}-go-build,target=/root/.cache/go-build \
    go build -trimpath -mod=readonly -buildvcs=false -ldflags="-s -w" \
    -o /out/secret-api ./cmd/server

# runtime
FROM gcr.io/distroless/base:nonroot

WORKDIR /app

COPY --from=builder --chown=nonroot:nonroot /out/secret-api /app/secret-api
COPY --from=builder --chown=nonroot:nonroot /src/web/static /app/web/static
COPY --from=builder --chown=nonroot:nonroot /src/web/robots.txt /app/web/robots.txt

EXPOSE 8080
ENV PORT=8080

ENTRYPOINT ["/app/secret-api"]
