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
├── config/
│   ├── app-config.yaml               # Primary dynamic configuration file
├── README.md
├── docker-compose.yml
├── go.mod                            # Module definition
├── go.sum
├── main.go                           # Entry point
├── pipelines/
│   └── benthos/
│       └── pipeline.yaml
├── src/
|   ├── config/
│   │   └── config_test.go            # Configuration Unit Tests
│   │   └── config.go                 # Configuration logic
│   ├── consumer/
│   │   └── consumer_interface.go     # Consumer Interface
│   │   └── consumer_test.go          # Consumer Unit Tests
│   │   └── consumer.go               # Consumer logic
|   ├── execute_pipeline/
│   │   └── execute_pipeline_test.go  # Pipeline Execution Unit Tests
│   │   └── execute_pipeline.go       # Pipeline Execution logic
|   ├── orchestrator/
│   │   └── orchestrator.go           # Pipeline lifecycle logic to manage setup, execution, monitoring, and shutdown.
|   ├── parsers/
│   │   └── json_parser.go            # JSON Parser logic
│   │   └── parsers_test.go           # Parsers Unit Tests
│   │   └── parsers.go                # Parsers logic
│   ├── producer/
│   │   └── producer_test.go          # Producer Unit Tests
│   │   └── producer.go               # Producer logic
├── docs/                             # images and project documentation
│── tests/                            # e2e and integration tests
│   |── integration/
│   │   ├── configs/                     # Integration test pipeline config
│   │   │   └── test-app-config.yaml     # Config file specific to integration testing
│   │   ├── pipelines/
│   │   │   └── test-pipeline.yaml       # Integration test dynamic pipeline generator template
│   │   └── test_data/
│   │   │   └── test-messages.json       # Example JSON files for test data
│   │   ├── docker-compose.override.yml  # Integration test-specific Docker Compose
│   │   ├── Dockerfile                   # Definition of the IT build image for the service
│   │   ├── helpers.go                   # Shared helper functions for integration tests
│   │   ├── integration_test.go          # Go test file for integration tests

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
  profiling: false
  pipeline_template: "pipelines/benthos/pipeline.yaml"
  generated_pipeline_config: "pipelines/benthos/generated-pipeline.yaml"
  source:
    parser: "json"               # Supported parsers: json, avro, parquet
    file: "data/input.json"      # Path to the input file or source
  kafka:
    brokers: ["localhost:9092"]
    topics: ["pipeline-topic"]
    consumer_group: "pipeline-group"
  postgres:
    url: "postgresql://admin:password@localhost:5432/pipelines?sslmode=disable"
    table: "processed_data"
  logger:
    level: "DEBUG"

```

1.	Save this configuration as config/app-config.yaml.
2.	Set the CONFIG_PATH environment variable:

```zsh
export CONFIG_PATH=config/app-config.yaml
```

**Example JSON Source File (messages.json)**
```json
{
  "messages": [
    "Message from JSON 1",
    "Message from JSON 2",
    "Message from JSON 3",
    "Message from JSON 4",
    "Message from JSON 5",
    "Message from JSON 6",
    "Message from JSON 7"
  ]
}
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
make integration-build
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

The data pipeline follows this workflow:

```plaintext
Source (JSON) → Kafka → Processors → Outputs (Postgres, Kafka, Logs)
```

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
    - Ensure the JSON file path in the configuration is correct.
    - Check Kafka logs for errors.

4.	**Metrics Not Visible:**
    - Ensure Prometheus PushGateway is running on port 9091.

---

# Contributing

Contributions are welcome! Please fork the repository, create a feature branch, and submit a pull request for review.

---

# License

Distributed Data Pipeline Manager is licensed under the MIT License. See the LICENSE file for details.
