# frontend build
FROM node:20-alpine@sha256:09e2b3d9726018aecf269bd35325f46bf75046a643a66d28360ec71132750ec8 AS frontend-builder

WORKDIR /src/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web .
RUN npm run build

# backend build
FROM golang:1.24-alpine@sha256:8bee1901f1e530bfb4a7850aa7a479d17ae3a18beb6e09064ed54cfd245b7191 AS builder

ENV CGO_ENABLED=0 GOOS=linux GO111MODULE=on

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

# Copy source code (excluding frontend which is copied separately)
COPY cmd ./cmd
COPY internal ./internal

# Copy frontend build output
COPY --from=frontend-builder /src/web/static/dist ./web/static/dist

# Copy web assets needed at runtime
COPY web/robots.txt ./web/robots.txt

RUN go build -trimpath -mod=readonly -buildvcs=false -ldflags="-s -w" \
    -o /out/secret-api ./cmd/server

# runtime
FROM gcr.io/distroless/base:nonroot@sha256:746b9dbe3065a124395d4a7698241dbd6f3febbf01b73e48f942aabd7b8e5eac

WORKDIR /app

COPY --from=builder --chown=nonroot:nonroot /out/secret-api /app/secret-api
COPY --from=builder --chown=nonroot:nonroot /src/web/static /app/web/static
COPY --from=builder --chown=nonroot:nonroot /src/web/robots.txt /app/web/robots.txt

EXPOSE 8080
ENV PORT=8080

ENTRYPOINT ["/app/secret-api"]
