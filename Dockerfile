# frontend build
FROM node:20-alpine AS frontend-builder

WORKDIR /src/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web .
RUN npm run build

# backend build
FROM golang:1.24-alpine AS builder

ENV CGO_ENABLED=0 GOOS=linux GO111MODULE=on

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# copy frontend
COPY --from=frontend-builder /src/web/static/dist /src/web/static/dist

RUN go build -trimpath -mod=readonly -buildvcs=false -ldflags="-s -w" \
    -o /out/secret-api ./cmd/server

# runtime
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=builder --chown=nonroot:nonroot /out/secret-api /app/secret-api
COPY --from=builder --chown=nonroot:nonroot /src/web/static /app/web/static
COPY --from=builder --chown=nonroot:nonroot /src/web/robots.txt /app/web/robots.txt

EXPOSE 8080
ENV PORT=8080

ENTRYPOINT ["/app/secret-api"]
