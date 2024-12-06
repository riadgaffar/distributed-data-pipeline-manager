# Build stage
FROM golang:1.22-bullseye AS builder

WORKDIR /app

# Install essential build dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    librdkafka-dev \
    libssl-dev \
    libsasl2-dev \
    libzstd-dev \
    zlib1g-dev \
    wget

# Install Redpanda
RUN curl -1sLf 'https://dl.redpanda.com/nzc4ZYQK3WRGd9sy/redpanda/cfg/setup/bash.deb.sh' | bash && \
    apt-get update && \
    apt-get install -y redpanda && \
    rpk --version

# Copy source code
COPY . .

# Build the application
RUN go mod download && \
    go mod tidy && \
    go build -o bin/pipeline_manager cmd/pipeline-manager/main.go

# Runtime stage
FROM debian:bullseye-slim

WORKDIR /app

# Install runtime dependencies and Redpanda
RUN apt-get update && \
    apt-get install -y librdkafka1 wget gnupg curl && \
    curl -1sLf 'https://dl.redpanda.com/nzc4ZYQK3WRGd9sy/redpanda/cfg/setup/bash.deb.sh' | bash && \
    apt-get update && \
    apt-get install -y redpanda && \
    rpk --version && \
    rm -rf /var/lib/apt/lists/*

# Copy binary and configuration files
COPY --from=builder /app/bin/pipeline_manager .
COPY pipelines ./pipelines
COPY config/app-config.yaml /app/config/app-config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/pipeline_manager"]