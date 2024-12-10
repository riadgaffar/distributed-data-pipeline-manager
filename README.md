# **Distributed Data Pipeline Manager**

The Distributed Data Pipeline Manager is a robust tool for managing, orchestrating, and monitoring data pipelines. It integrates with Redpanda (Kafka alternative), processes messages using Benthos, and outputs results to PostgreSQL. This tool includes dynamic configuration support, profiling, and integration testing capabilities.

---

## **Features**

- **Dynamic Pipeline Configuration**: Define pipelines with support for multiple input, processing, and output stages.
- **Orchestration**: Use an orchestrator for seamless pipeline management with dynamic configuration and timeout support.
- **Profiling**: Built-in CPU and memory profiling for performance analysis.
- **Integration Testing**: Dedicated integration test framework with environment variable control.
- **Monitoring and Logging**: Integrated health checks, Prometheus metrics, and configurable logging.

---

## **Getting Started**

### Prerequisites

- **Go** (version 1.22 or later)
- **Docker** (with Docker Compose)
- **Redpanda CLI (`rpk`)**
  `brew install redpanda-data/tap/redpanda`
- **YAML Validator**: Install `yamllint` for validating YAML files.
  `brew install yamllint`

---

# Project structure
```plaintext
distributed-data-pipeline-manager/
├── cmd/                              # Main application entry points
│   ├── pipeline-manager/             # Entry point for the pipeline manager
│   │   └── main.go                   # Main application logic
├── config/
│   ├── app-config.yaml               # Primary dynamic configuration file
├── deployments/                      # Deployment configurations
│   ├── docker/                       # Docker-related deployment files
│   │   ├── Dockerfile                # Build instructions for the app
│   │   ├── docker-compose.yml        # Docker Compose for local deployment
│   └── k8s/                          # Kubernetes manifests and Helm charts
│       ├── base/                     # Raw Kubernetes YAML files
│       │   ├── deployment.yaml       # App deployment manifest
│       │   ├── service.yaml          # App service manifest
│       │   ├── configmap.yaml        # App configuration as a ConfigMap
│       │   ├── secret.yaml           # Secrets for sensitive data
│       ├── overlays/                 # Environment-specific customizations
│           ├── dev/                  # Development environment configs
│           ├── prod/                 # Production environment configs
├── docs/                            # Documentation for the project
│   ├── images/                      # Images for README and docs
├── pipelines/                       # Pipeline templates and configs
│   └── benthos/
│       └── pipeline.yaml
├── src/
│   ├── bootstrap/                   # Application initialization and setup logic
│   │   ├── bootstrap.go             # Initialization logic for app, Kafka, and parser
│   │   └── bootstrap_test.go        # Unit tests for bootstrap logic
│   ├── config/                      # Configuration management
│   │   ├── config.go                # Configuration loading and validation logic
│   │   └── config_test.go           # Configuration unit tests
│   ├── consumer/                    # Consumer logic and interfaces
│   │   ├── consumer.go              # Consumer implementation
│   │   ├── consumer_interface.go    # Consumer interface
│   │   └── consumer_test.go         # Consumer unit tests
│   ├── execute_pipeline/            # Logic for pipeline execution
│   │   ├── execute_pipeline.go      # Pipeline execution logic
│   │   └── execute_pipeline_test.go # Unit tests for pipeline execution
│   ├── monitoring/                  # Monitoring and scaling logic
│   │   ├── monitoring.go            # Monitoring for Kafka lag and scaling
│   │   └── monitoring_test.go       # Unit tests for monitoring logic
│   ├── orchestrator/                # Pipeline lifecycle orchestration
│   │   └── orchestrator.go          # Orchestrator logic
│   ├── parsers/                     # Parsers for input data
│   │   ├── parsers.go               # Generic parser interface and logic
│   │   ├── json_parser.go           # JSON parser logic
│   │   └── parsers_test.go          # Unit tests for parsers
│   ├── producer/                    # Kafka producer and topic management
│   │   ├── producer.go              # Producer logic for Kafka
│   │   ├── producer_test.go         # Producer unit tests
│   │   └── kafka.go                 # Kafka-specific partition management
│   ├── server/                      # HTTP server for health checks and metrics
│   │   ├── server.go                # HTTP server implementation
│   │   └── server_test.go           # HTTP server unit tests
│   ├── utils/                       # Utility functions and shared logic
│   │   ├── signals.go               # Signal handling for graceful shutdown
│   │   ├── signals_test.go          # Unit tests for signal handling
│   │   ├── profiling.go             # Profiling utilities
│   │   ├── profiling_test.go        # Profiling unit tests
│   │   ├── logger.go                # Logging configuration
│   │   └── logger_test.go           # Unit tests for logging
├── tests/                           # e2e and integration tests
│   ├── integration/                 # Integration test utilities and examples
│   │   ├── configs/                 # Integration test pipeline config
│   │   │   └── test-app-config.yaml # Config file specific to integration testing
│   │   ├── pipelines/
│   │   │   └── test-pipeline.yaml   # Integration test dynamic pipeline generator template
│   │   ├── plugins/
│   │   │   └── json/                # JSON format tests
│   │   │       └── json_parser_test.go
│   │   │   └── avro/                # Avro format tests
│   │   │   └── parquet/             # Parquet format tests
│   │   │   └── custom/              # Custom format tests
│   │   └── test_data/
│   │       └── test-messages.json   # Example JSON files for test data
│   │   ├── docker-compose.test.yml  # Integration test-specific Docker Compose
│   │   ├── Dockerfile               # Definition of the IT build image for the service
│   │   ├── helpers.go               # Shared helper functions for integration tests
│   │   └── integration_test.go      # Go test file for integration tests
├── go.mod                           # Go module definition
├── go.sum                           # Go dependencies checksum
├── Makefile                         # Common commands for building, testing, running
├── LICENSE                          # License file
├── README.md                        # Main documentation
```

---

# Setup

### Clone the Repository

```zsh
git clone https://github.com/your-username/distributed-data-pipeline-manager.git
cd distributed-data-pipeline-manager
```

### Install Dependencies

```zsh
go mod tidy
```

## Configuration

The application is configured using an environment variable, CONFIG_PATH, which points to a dynamic configuration file (app-config.yaml). Example:

**Example Configuration File (app-config.yaml)**
```yaml
  app:
    profiling: false                 # Enable for development if needed
    pipeline_template: "pipelines/benthos/pipeline.yaml"
    generated_pipeline_config: "pipelines/benthos/generated-pipeline.yaml"
    source:
      parser: "json"                 # Future Supported parsers: json, avro, parquet
      plugin_path: "/app/bin/plugins/json.so"
    kafka:
      brokers: ["redpanda:9092"]     # Use the hostname of the Kafka broker
      topics: ["pipeline-topic"]
      consumer_group: "pipeline-group"
      min_partitions: 3
    postgres:
      url: "postgresql://admin:password@postgres:5432/pipelines?sslmode=disable"
      table: "processed_data"
    logger:
      level: "DEBUG"
```

1.	Save this configuration as config/app-config.yaml.
2.	Set the CONFIG_PATH environment variable:

```zsh
export CONFIG_PATH=config/app-config.yaml
```

### Orchestrator

The orchestrator coordinates the pipeline execution. It dynamically adjusts based on the environment:

	- Integration Test Mode: Controlled by the INTEGRATION_TEST_MODE environment variable.

	- Timeout Support: Automatically stops the pipeline after a defined duration in test mode.

Orchestrator Usage in Code

```go
isTesting := os.Getenv("INTEGRATION_TEST_MODE") == "true"
timeout := 0 * time.Second
if isTesting {
    timeout = 30 * time.Second // Adjust for integration tests
}
orchestrator := orchestrator.NewOrchestrator(cfg, &execute_pipeline.RealCommandExecutor{}, isTesting, timeout)

if err := orchestrator.Run(); err != nil {
    log.Fatalf("ERROR: Orchestrator encountered an issue: %v\n", err)
}
```

---

# Unit Tests

```zsh
make test
```

# Integration Tests

```zsh
make test-integration-plugin format=json  # Or avro, parquet, not available yet
```

---

# Running the Application

## Steps to Run

**1.	Build and Start Services:**
Use the Makefile to build and run the application with Docker Compose:

```zsh
make run
```

**2.	Monitor Logs:**
Verify the logs to confirm pipeline execution:

```plaintext
INFO: Distributed Data Pipeline Manager
INFO: Loaded configuration: {...}
```

**3.	Shutdown:**
Use CTRL+C to gracefully stop the application. Profiling data will be saved if enabled.

**4.	Clean Up:**
Use make reset to stop services and clean up containers:

```zsh
make reset
```

## Pipeline Workflow

### Overview

The data pipeline follows this data processing workflow:

```plaintext
Source (JSON | AVRO | PARQUET) → Kafka → Processors → Outputs (Postgres, Kafka, Logs)
```
![Memory Graph](docs/images/ddpm_wf.png)

**Example Pipeline Configuration (Generated at Runtime)**
```yaml
# Input section: Consumes data from Kafka
input:
  kafka:
    addresses: ["localhost:9092"]
    topics: ["test-topic"]
    consumer_group: "test-group"
    checkpoint_limit: 1000

# Pipeline section: Processes and transforms data
pipeline:
  processors:
    - mapping: |
        root.id = uuid_v4()
        root.timestamp = now()
        root.data = content().uppercase()

# Output section: Routes data to multiple destinations
output:
  broker:
    outputs:
      - sql_insert:
          driver: postgres
          dsn: "postgresql://admin:password@localhost:5432/pipelines?sslmode=disable"
          table: "processed_data"
          columns: ["id", "timestamp", "data"]
          args_mapping: |
            root = [
              this.id,
              this.timestamp,
              this.data
            ]
          max_in_flight: 5

      - kafka:
          addresses: ["localhost:9092"]
          topic: "failed-messages"
          compression: gzip

      - stdout:
          codec: lines
```

---

# Profiling

Profiling helps monitor performance by generating CPU and memory usage data.

## Enabling Profiling

	•	Enable Profiling: Set profiling: true in app-config.yaml.
	•	Run the Application: Profiling data is generated when the app exits (cpu.pprof, mem.pprof).

## Analyzing Profiling Data

```zsh
go tool pprof -http=:8080 cpu.pprof
```

## Monitoring

The application provides the following endpoints:
	•	**Health Check:** http://localhost:4195/health
	•	**Metrics:** http://localhost:4195/metrics

For metrics aggregation, set up Prometheus PushGateway.

**Prerequisite: `graphviz`**

```zsh
brew install graphviz
```

**This uses runtime/pprof to programmatically collect profiles and will generate `cpu.pprof` and `mem.pprof` files. Run the following commands to generate visual CPU and Memory graphs.**

```zsh
go tool pprof -png ./bin/pipeline_manager cpu.pprof > ./docs/images/cpu.png
go tool pprof -png ./bin/pipeline_manager mem.pprof > ./docs/images/mem.png
```

## CPU Graph:
![CPU Graph](docs/images/cpu.png)

## Memory Graph:

![Memory Graph](docs/images/mem.png)

---

# Troubleshooting

## Common Issues

1.	**Configuration Errors:**
    - Ensure CONFIG_PATH is set correctly.
    - Validate YAML syntax using yamllint.

2.	**Kafka or Postgres Connection Issues:**
    - Confirm services are running and accessible.
    - Verify the kafka.brokers and postgres.url settings.

3.	**No Messages Processed:**
    - Check Kafka logs for errors.

4.	**Metrics Not Visible:**
    - Ensure Prometheus PushGateway is running on port 9091.

---

# Contributing

Contributions are welcome! Please fork the repository, create a feature branch, and submit a pull request for review.

---

# License

Distributed Data Pipeline Manager is licensed under the MIT License. See the LICENSE file for details.
