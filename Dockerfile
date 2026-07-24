# syntax=docker/dockerfile:1.6

# Build stage runs natively on the build host and cross-compiles to the target
# platform. glebarez/sqlite is pure Go (no CGO), so this is a plain cross-compile.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder
RUN apk add --no-cache esbuild
WORKDIR /src
COPY . .
RUN go generate ./internal/frontend/...

ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/rendezvous ./cmd/rendezvous

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S rendezvous && adduser -S rendezvous -G rendezvous
WORKDIR /app
COPY --from=builder --chown=rendezvous:rendezvous /out/rendezvous .
# Writable directory for the default sqlite DSN (data/rendezvous.db).
RUN mkdir -p /app/data && chown -R rendezvous:rendezvous /app
USER rendezvous
EXPOSE 8080
ENTRYPOINT ["./rendezvous"]
