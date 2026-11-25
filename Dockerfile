ARG GO_VERSION=1.26

FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -o /out/control-plane \
    ./cmd/api

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -o /out/worker \
    ./cmd/worker


# Shared runtime image
FROM alpine:3.23 AS runtime

RUN apk add --no-cache ca-certificates curl \
    && addgroup -S -g 10001 app \
    && adduser -S -D -H -u 10001 -G app app \
    && mkdir -p /var/lib/tenant-worker \
    && chown app:app /var/lib/tenant-worker

WORKDIR /app

USER app


# Control Plane image
FROM runtime AS control-plane

COPY --from=builder \
    /out/control-plane \
    /usr/local/bin/control-plane

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/control-plane"]


# Worker image
FROM runtime AS worker

COPY --from=builder \
    /out/worker \
    /usr/local/bin/worker

ENTRYPOINT ["/usr/local/bin/worker"]