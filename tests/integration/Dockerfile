FROM golang:1.22-bullseye AS builder

# Install dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    pkg-config \
    libssl-dev \
    libsasl2-dev \
    libzstd-dev \
    zlib1g-dev \
    librdkafka-dev \
    wget

# Set up the Go application
WORKDIR /app
COPY . .
COPY pipelines /app/pipelines
COPY tests/integration/configs/test-app-config.yaml /app/config.yaml

# Install confluent-kafka-go dynamically linked to the system's librdkafka
RUN go get -tags dynamic github.com/confluentinc/confluent-kafka-go/v2@v2.6.1
RUN go mod tidy

# Command to run integration tests
CMD ["go", "test", "-v", "./tests/integration/..."]