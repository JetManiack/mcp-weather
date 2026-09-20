# syntax=docker/dockerfile:1.6

# Build stage runs natively on the build host and cross-compiles to the target
# platform. glebarez/sqlite is pure Go (no CGO), so this is a plain cross-compile.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder
RUN apk add --no-cache ca-certificates make curl openssl bash nodejs npm
# Install a pinned esbuild that matches the version in package.json devDependencies.
RUN npm install --global esbuild@0.28.1
WORKDIR /src
COPY . .
RUN make generate
RUN [ -s internal/frontend/static/js/app.bundle.js ] || { echo "bundle missing or empty"; exit 1; }

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=docker
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/weather ./cmd/weather

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S weather && adduser -S weather -G weather
WORKDIR /app
COPY --from=builder --chown=weather:weather /out/weather .
RUN mkdir -p /app/data && chown -R weather:weather /app
USER weather
EXPOSE 8080
ENTRYPOINT ["./weather"]
