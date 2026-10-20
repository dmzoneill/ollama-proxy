# Multi-stage build for ollama-proxy
FROM golang:1.24 AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with version information
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o ollama-proxy \
    ./cmd/proxy

# Final stage - minimal image. Pin the release tag and apk package
# versions so image rebuilds do not float.
FROM alpine:3.24.2

# Install CA certificates, then create the unprivileged user and config directory.
RUN apk --no-cache add ca-certificates=20260909-r0 \
    && addgroup -S ollama \
    && adduser -S ollama -G ollama \
    && mkdir -p /etc/ollama-proxy/config \
    && chown -R ollama:ollama /etc/ollama-proxy

# Copy binary from builder
COPY --from=builder /app/ollama-proxy /usr/local/bin/ollama-proxy

# Copy default config (optional - can be mounted)
COPY --from=builder /app/config/config.yaml /etc/ollama-proxy/config/config.yaml

# Switch to non-root user
USER ollama

# Expose ports
EXPOSE 8080 50051 9090

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/ollama-proxy"]
CMD ["--config", "/etc/ollama-proxy/config/config.yaml"]
