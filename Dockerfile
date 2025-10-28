# build
FROM golang:1.24-alpine AS builder

ENV CGO_ENABLED=0 GOOS=linux GO111MODULE=on

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -trimpath -mod=readonly -buildvcs=false -ldflags="-s -w" \
    -o /out/secret-api ./cmd/server

# runtime
FROM gcr.io/distroless/base:nonroot

WORKDIR /app

COPY --from=builder --chown=nonroot:nonroot /out/secret-api /app/secret-api
COPY --from=builder --chown=nonroot:nonroot /src/web /app/web

USER 65532:65532

EXPOSE 8080
ENV PORT=8080

ENTRYPOINT ["/app/secret-api"]
