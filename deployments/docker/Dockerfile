# Build stage
FROM golang:1.21-bookworm AS builder

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y \
    git \
    && rm -rf /var/lib/apt/lists/*

# Copy source code
COPY . .

# Generate go.sum with all dependencies (including indirect)
RUN go mod tidy

# Build application
RUN CGO_ENABLED=0 GOOS=linux go build -o /obsidian-server ./cmd/vpn-server

# Final stage
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    wireguard-tools \
    iptables \
    iproute2 \
    procps \
    curl \
    ca-certificates \
    bash \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary from builder
COPY --from=builder /obsidian-server /app/obsidian-server

# Copy scripts
COPY deployments/scripts/entrypoint.sh /app/entrypoint.sh
COPY deployments/scripts/setup-wg.sh /app/setup-wg.sh

RUN chmod +x /app/entrypoint.sh /app/setup-wg.sh

# Expose ports
EXPOSE 8080
EXPOSE 51820/udp

ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["/app/obsidian-server"]
