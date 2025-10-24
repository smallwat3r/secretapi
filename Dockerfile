FROM golang:1.24-alpine AS builder

ENV CGO_ENABLED=0 GO111MODULE=on

RUN apk add --no-cache git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -trimpath -ldflags="-s -w" -o /out/secret-api ./cmd/...

FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=builder /out/secret-api /app/secret-api

EXPOSE 8080

USER nonroot:nonroot

ENV PORT=8080

ENTRYPOINT ["/app/secret-api"]
