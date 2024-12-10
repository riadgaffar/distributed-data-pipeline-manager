FROM golang:1.23.4-bullseye AS builder

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

# Install rpk
RUN curl -1sLf \
    'https://dl.redpanda.com/nzc4ZYQK3WRGd9sy/redpanda/cfg/setup/bash.deb.sh' | bash && \
    apt-get update && \
    apt-get install -y redpanda
    
RUN rpk --version

WORKDIR /app
COPY . .
COPY pipelines /app/pipelines
COPY tests/integration/pipelines/test-pipeline.yaml /app/pipelines/
RUN go get -tags dynamic github.com/confluentinc/confluent-kafka-go/v2@v2.6.1
RUN go mod tidy

CMD ["tail", "-f", "/dev/null"]
