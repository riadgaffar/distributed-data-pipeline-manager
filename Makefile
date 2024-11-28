# Variables
APP_NAME := pipeline_manager
SRC_DIR := src
BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)
DOCKER_COMPOSE := docker-compose.yml
POSTGRES_DATA := postgres-data
IT_POSTGRES_DATA := tests/integration/postgres-data
REDPANDA_DATA := redpanda-data
IT_REDPANDA_DATA := tests/integration/redpanda-data
CONFIG_PATH=config/app-config.yaml
WORK_DIR := $(shell dirname $(realpath main.go))
COMPOSE_FILES := -f docker-compose.yml -f ./tests/integration/docker-compose.override.yml

# Default target
.PHONY: all
all: build

# Build the Go binary
.PHONY: build
build:
	@echo "Building the application..."
	mkdir -p $(BIN_DIR)
	cd $(WORK_DIR) && go build -o $(BIN_PATH) main.go
	@echo "Build complete: $(BIN_PATH)"

# Build the Go binary with debug flags
.PHONY: build-debug
build-debug:
	@echo "Building the application with debug flags..."
	mkdir -p $(BIN_DIR)
	cd $(WORK_DIR) && go build -gcflags="all=-N -l" -o $(BIN_PATH) main.go
	@echo "Build complete: $(BIN_PATH)"
	
# Run the application
.PHONY: run
run: build docker-up
	@echo "Running the application..."
	cd $(WORK_DIR) && CONFIG_PATH=$(CONFIG_PATH) ./$(BIN_PATH)

# Run the application with profiling
.PHONY: run-profile
run-profile: build docker-up
	@echo "Running the application with profiling enabled..."
	cd $(WORK_DIR) && CONFIG_PATH=$(CONFIG_PATH) ./$(BIN_PATH) -profile=true

# Debug the application using Delve
.PHONY: debug
debug: build-debug
	@echo "Starting debug mode..."
	cd $(WORK_DIR) && dlv exec ./$(BIN_PATH)

# Run Docker Compose
.PHONY: docker-up
docker-up:
	@echo "Starting Docker Compose services..."
	docker compose -f $(DOCKER_COMPOSE) up -d

# Stop Docker Compose
.PHONY: docker-down
docker-down:
	@echo "Stopping and removing all containers..."
	docker compose -f $(DOCKER_COMPOSE) down -v

# Clean Docker Build Cache
.PHONY: docker-clean-cache
docker-clean-cache:
	@echo "Cleaning docker build cache..."
	docker builder prune -f
	@echo "Docker build cache clean complete."

# Clean up data artifacts
.PHONY: data-clean
data-clean:
	@echo "Cleaning up data artifacts..."
	rm -rf $(POSTGRES_DATA)
	rm -rf $(REDPANDA_DATA)
	rm -fr $(IT_POSTGRES_DATA)
	rm -fr $(IT_REDPANDA_DATA)
	@echo "Data clean complete."

# Clean up build artifacts
.PHONY: clean
clean:
	@echo "Cleaning up build artifacts..."
	rm -rf $(BIN_DIR)
	@echo "Binary clean complete."

# Clean up integration test containers and volumes
.PHONY: integration-clean
integration-clean:
	@echo "Stopping and cleaning up integration test containers and volumes..."
	docker compose -f tests/integration/docker-compose.override.yml down -v --remove-orphans
	@echo "Integration test environment cleaned up."

# Remove all Kafka topics
.PHONY: rm-kafka-topics
rm-kafka-topics:
	@echo "Deleting all Kafka topics..."
	rpk topic list | awk 'NR>1 {print $$1}' | xargs -I {} rpk topic delete {}

# Remove generated pipeline config
.PHONY: rm-generated-pipeline-config
rm-generated-pipeline-config:
	@echo "Removing generated pipeline files..."
	rm -f pipelines/benthos/generated-pipeline.yaml

# Full reset (clean + docker down)
.PHONY: reset
reset: clean docker-clean-cache docker-down rm-kafka-topics data-clean rm-generated-pipeline-config integration-clean
	@echo "Project reset complete."

# Debug target to print paths and environment info
.PHONY: debug-info
debug-info:
	@echo "Current working directory: $(WORK_DIR)"
	@echo "Binary path: $(BIN_PATH)"
	@echo "Source directory: $(SRC_DIR)"
	@echo "Docker Compose file: $(DOCKER_COMPOSE)"

# .PHONY: integration-test
# integration-test:
# 	@echo "Running integration tests..."
# 	docker compose --profile testing -f tests/integration/docker-compose.override.yml build
# 	docker compose --profile testing -f tests/integration/docker-compose.override.yml up -d
# 	docker compose -f tests/integration/docker-compose.override.yml logs -f &
# 	@echo "Waiting for services to become ready..."
# 	@docker compose -f tests/integration/docker-compose.override.yml wait test-pipeline-manager || (echo "Services failed to become ready" && exit 1)
# 	@docker compose -f tests/integration/docker-compose.override.yml exec test-pipeline-manager go test -v ./tests/integration/...
# 	@docker compose -f tests/integration/docker-compose.override.yml down -v --remove-orphans
# 	@echo "Integration tests completed successfully."

# Run Go tests
.PHONY: test
test:
	@echo "Running tests..."
	cd $(WORK_DIR) && go test -v ./...
	@echo "Test run complete."
