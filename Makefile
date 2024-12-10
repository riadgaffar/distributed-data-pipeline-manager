# Variables
PROJECT_ROOT := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
APP_NAME := pipeline_manager
SRC_DIR := src
PLUGINS_SRC_DIR := $(PROJECT_ROOT)/plugins
CMD_DIR := $(PROJECT_ROOT)/cmd/pipeline-manager
BIN_DIR := $(PROJECT_ROOT)/bin
PLUGINS_BIN_DIR := $(PROJECT_ROOT)/bin/plugins
BIN_PATH := $(BIN_DIR)/$(APP_NAME)
DOCKER_COMPOSE := $(PROJECT_ROOT)/deployments/docker/docker-compose.yml
POSTGRES_DATA := $(PROJECT_ROOT)/deployments/docker/postgres-data
REDPANDA_DATA := $(PROJECT_ROOT)/deployments/docker/redpanda-data
CONFIG_PATH := $(PROJECT_ROOT)/config/app-config.yaml
IT_POSTGRES_DATA := tests/integration/postgres-data
IT_REDPANDA_DATA := tests/integration/redpanda-data
TEST_DIRS := $(shell go list ./... | grep -v 'tests/integration')

# Default target
.PHONY: all
all: build

# Build the Go plugins
.PHONY: build-plugins
build-plugins:
	@echo "Building parser plugins..."
	mkdir -p $(PLUGINS_BIN_DIR)
	@echo "Building parser plugins..."
	CGO_ENABLED=1 go build -buildmode=plugin -o $(PROJECT_ROOT)/bin/plugins/json.so $(PLUGINS_SRC_DIR)/json/json_parser.go
	@echo "Build complete: $(PLUGINS_BIN_DIR)"

# Build the Go binary
.PHONY: build
build:
	@echo "Building the application..."
	mkdir -p $(BIN_DIR)	
	cd $(CMD_DIR) && go build -o $(BIN_PATH) main.go
	@echo "Build complete: $(BIN_PATH)"

# Build the Go binary with debug flags
.PHONY: build-debug
build-debug:
	@echo "Building the application with debug flags..."
	mkdir -p $(BIN_DIR)
	cd $(CMD_DIR) && go build -gcflags="all=-N -l" -o $(BIN_PATH) main.go
	@echo "Build complete: $(BIN_PATH)"

# Run the application
.PHONY: run
run: docker-up
	@if docker ps | grep $(APP_NAME); then \
		echo "Application stack is running..."; \
	else \
		echo "Application stack has stopped."; \
	fi

# Run the application with profiling
.PHONY: run-profile
run-profile: build docker-up
	@echo "Running the application with profiling enabled..."
	CONFIG_PATH=$(CONFIG_PATH) $(BIN_PATH) -profile=true

# Debug the application using Delve
.PHONY: debug
debug: build-debug
	@echo "Starting debug mode..."
	dlv exec ./$(BIN_PATH)

# Run Docker Compose
.PHONY: docker-up
docker-up:
	@echo "Starting Docker Compose services..."
	docker compose -f $(DOCKER_COMPOSE) build --no-cache
	docker compose -f $(DOCKER_COMPOSE) up --build --abort-on-container-exit || { \
		status=$$?; \
		if [ $$status -eq 2 ]; then \
			echo "INFO: Containers exited as expected."; \
			exit 0; \
		fi; \
		exit $$status; \
	}

# Stop Docker Compose
.PHONY: docker-down
docker-down:
	@echo "Stopping and removing all containers..."
	docker compose -f $(DOCKER_COMPOSE) down -v

.PHONY: logs
logs:
	@echo "Streaming Docker Compose logs..."
	docker compose -f $(DOCKER_COMPOSE) logs --follow

# Clean Docker Networks
.PHONY: docker-clean-networks
docker-clean-networks:
	@echo "Cleaning docker build networks..."
	docker network prune -f 
	@echo "Docker build networks clean complete."

# Clean Docker Build Cache
.PHONY: docker-clean-cache
docker-clean-cache:
	@echo "Cleaning docker build cache..."
	docker builder prune --all -f
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
	go clean -cache 
	go clean -modcache 
	go clean -testcache
	@echo "Binary clean complete."

# Debug target to print paths and environment info
.PHONY: debug-info
debug-info:
	@echo "Project root: $(PROJECT_ROOT)"
	@echo "Binary path: $(BIN_PATH)"
	@echo "Docker Compose file: $(DOCKER_COMPOSE)"

.PHONY: integration-test
integration-test:
	@echo "Running integration tests..."
	# Build the Docker images for integration testing
	docker compose --profile testing -f tests/integration/docker-compose.override.yml build
	
	# Start the containers and run the tests, exiting with the code of test-pipeline-manager
	docker compose --profile testing -f tests/integration/docker-compose.override.yml up --exit-code-from test-pipeline-manager
	
	# Capture logs (optional)
	docker compose -f tests/integration/docker-compose.override.yml logs
	
	@echo "Integration tests completed successfully."

# Run Go tests
.PHONY: test
test:
	@echo "Running unit tests..."
	cd $(CMD_DIR) && go test -v $(TEST_DIRS)
	@echo "Unit test run complete."

# Clean up integration test containers and volumes
.PHONY: integration-clean
integration-clean:
	@echo "Stopping and cleaning up integration test containers and volumes..."
	docker compose -f tests/integration/docker-compose.override.yml down -v --remove-orphans
	@if [ -n "$$(docker ps -aq --filter 'status=exited')" ]; then \
		docker rm $$(docker ps -aq --filter 'status=exited'); \
	fi
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
	rm -f tests/integration/pipelines/test-generated-pipeline.yaml

# Full reset (clean + docker down)
.PHONY: reset
reset: clean docker-down rm-kafka-topics data-clean rm-generated-pipeline-config integration-clean docker-clean-networks docker-clean-cache
	@echo "Project reset complete."
