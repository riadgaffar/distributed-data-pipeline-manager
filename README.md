# **Distributed Data Pipeline Manager**

A modern, flexible tool for managing distributed data pipelines using Kafka, Redpanda, Postgres, and dynamic configuration. The application supports message ingestion, transformation, and routing through customizable configurations.

---

## **Features**

- Dynamic configuration with YAML (pipeline setup) and JSON (message sources).
- Kafka/Redpanda for messaging and Postgres for storage.
- Customizable processing using mappings and transformations.
- Logging and profiling for monitoring performance.
- Health check and metrics endpoints for system observability.

---

## **Getting Started**

### Prerequisites

- **Docker**: Install Docker and Docker Compose.
- **Kafka CLI Tools**: Install Redpanda’s CLI (`rpk`).
  `brew install redpanda-data/tap/redpanda`
- **Go**: Install Go (v1.20 or later).  
- **YAML Validator**: Install `yamllint` for validating YAML files.
  `brew install yamllint`

### Clone the Repository

```zsh
git clone https://github.com/your-username/distributed-data-pipeline-manager.git
cd distributed-data-pipeline-manager
```

---

# Project structure
```bash
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
|   ├── parsers/
│   │   └── json_parser.go            # JSON Parser logic
│   │   └── parsers_test.go           # Parsers Unit Tests
│   │   └── parsers.go                # Parsers logic
│   ├── producer/
│   │   └── producer_test.go          # Producer Unit Tests
│   │   └── producer.go               # Producer logic
├── docs/                             # images and project documentation
└── tests/                            # e2e and integration tests
```

---

# Setup

## Configuration

The application uses dynamic configurations for customizing the pipeline behavior. Configurations are stored in a YAML file.

**Example Configuration File (app-config.yaml)**
```yaml
app:
  profiling: false
  source:
    parser: "json"
    file: "messages.json"
  kafka:
    brokers:
      - "localhost:9092"
    topics:
      - "test-topic"
    consumer_group: "test-group"
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

---

# Running Tests

```bash
make test
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

```bash
brew install graphviz
```

**This uses runtime/pprof to programmatically collect profiles and will generate `cpu.pprof` and `mem.pprof` files. Run the following commands to generate visual CPU and Memory graphs.**

```bash
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
