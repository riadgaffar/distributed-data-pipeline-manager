# **Distributed Data Pipeline Manager**

A robust tool for managing, orchestrating, and monitoring data pipelines. It integrates with Redpanda (Kafka alternative), processes messages using Benthos, and outputs results to PostgreSQL. Features include dynamic configuration, plugin support for multiple data formats, profiling, and comprehensive integration testing.

---

## **Features**

- **Dynamic Pipeline Configuration**: Define pipelines with multiple input, processing, and output stages
- **Plugin System**: Extensible parser plugins supporting JSON, Avro, and Parquet formats
- **Orchestration**: Pipeline management with dynamic configuration and timeout support
- **Profiling**: Built-in CPU and memory profiling
- **Integration Testing**: Dedicated framework with format-specific plugin testing
- **Monitoring**: Health checks, Prometheus metrics, and configurable logging

---

## Pipeline Workflow

### Overview

The data pipeline follows this data processing workflow:

```plaintext
IoT Sensors → JSON Messages → Redpanda → Pipeline Manager
                                         ├─→ PostgreSQL (Valid Data)
                                         ├─→ Error Topic (Invalid Data)
                                         └─→ Audit Logs (Compliance)
```

This architecture supports:

- High throughput with parallel processing
- Plugin-based format handling
- Real-time monitoring and alerting
- Data quality enforcement
- Audit trail maintenance

![Workflow](docs/images/ddpm_wf.png)

---

### Basic Metrics Overview

<img src="docs/images/ddpm_basic_metrics.png" width="800" height="400" alt="CPU Graph">

---

## **Getting Started**

### Prerequisites

```bash
# Install Go
brew install go@1.22

# Install Redpanda CLI
brew install redpanda-data/tap/redpanda

# Install YAML validator
brew install yamllint
```

---

# Project structure
```plaintext
distributed-data-pipeline-manager/
├── cmd/                             # Main application entry points
│   ├── pipeline-manager/            # Entry point for the pipeline manager
│   │   └── main.go                  # Main application logic
├── config/
│   ├── app-config.yaml              # Primary dynamic configuration file
├── deployments/                     # Deployment configurations
│   ├── docker/                      # Docker-related deployment files
│   │   ├── grafana-data             # Grafana dasboard configuration snap shot
│   │   ├── init-scripts             # Database sql seed scripts
│   │   ├── prometheus-config        # Prometheus config yaml
│   │   ├── Dockerfile               # Build instructions for the app
│   │   ├── docker-compose.yml       # Docker Compose for local deployment
├── docs/                            # Documentation for the project
│   ├── images/                      # Images for README and docs
├── pipelines/                       # Pipeline templates and configs
│   └── benthos/
│       └── pipeline.yaml
│   ├── plugins/
│   │   └── json/                    # JSON format plugin
│   │   └── avro/                    # Avro format plugin
│   │   └── parquet/                 # Parquet format plugin
│   │   └── custom/                  # Custom format plugin
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
│   │   ├── framework/               # Integration testing infrastructure for integration tests
│   │   │   └── assertion.go         # TestAssertions struct with methods for test validations and comparisons
│   │   │   └── base.go              # TestFramework struct that coordinates the testing components
│   │   │   └── helper.go            # PipelineTestHelper with core functionality
│   │   │   └── plugin.go            # PluginTest interface and BasePluginTest struct that plugin-specific tests implement
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
│   │   ├── run-tests.sh             # Script to run integration test
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

**Example Application Configuration File (app-config.yaml)**
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


## Environment Setup

1.	Save this configuration as config/app-config.yaml.
2.	Set the CONFIG_PATH environment variable:

```zsh
export CONFIG_PATH=config/app-config.yaml
```

## Orchestrator

The orchestrator coordinates pipeline execution with the following features:

### Core Features
- Dynamic configuration loading
- Pipeline lifecycle management
- Graceful shutdown handling
- Test mode support with timeouts
- Plugin system integration

### Configuration
```go
type Orchestrator struct {
    configLoader ConfigLoader
    config       *config.AppConfig
    executor     execute_pipeline.CommandExecutor
    producer     producer.Producer
    stopChan     chan os.Signal
    timeout      time.Duration
    isTesting    bool
}
```

### Usage Examples

**1. Production/Development Mode:**
```go
orchestrator := orchestrator.NewOrchestrator(
    configLoader,
    cfg,
    &execute_pipeline.RealCommandExecutor{},
    producer,
    false,  // not testing
    0,      // no timeout
)
```

**2. Test Mode:**
```go
orchestrator := orchestrator.NewOrchestrator(
    configLoader,
    cfg,
    &execute_pipeline.RealCommandExecutor{},
    producer,
    true,               // testing mode
    30 * time.Second,   // timeout
)
```

### Environment Variables

```zsh
INTEGRATION_TEST_MODE: Enable test mode
CONFIG_PATH: Path to test configuration file
```

---

# Plugin System

## Available Plugins

- JSON Parser: Default parser for JSON data
- Avro Parser: (Coming soon)
- Parquet Parser: (Coming soon)

## Creating Custom Plugins

1. Create plugin implementation:

```go
// plugins/custom/custom_parser.go
package main

type CustomParser struct{}

func (p *CustomParser) Parse(data []byte) (interface{}, error) {
    // Implementation
}
```

2. Build plugin:

```zsh
make build-plugins
```

---

# Running the Application

## LocalDevelopment

```zsh
# Start services
make run

# Monitor logs
docker compose logs -f

# Graceful shutdown
make reset
```

## Debug Mode

```zsh
# Build with debug flags
make build-debug

# Run with debugger
make debug
```

---

# Unit Tests

```zsh
make test
```

# Integration Tests

```zsh
# Test specific format
make test-integration-plugin format=json # Or avro, parquet, not available yet

# Clean test environment
make integration-clean
```

## Common Test Issues & Solutions

1. **Pipeline Not Starting**

```zsh
# Check service health
docker ps
docker logs test-pipeline-manager

# Verify network connectivity
docker network inspect test-network
```

2. **Message Processing Issues**

```zsh
# Check Kafka topics
docker exec test-redpanda rpk topic list
docker exec test-redpanda rpk topic consume test-topic

# Verify PostgreSQL data
docker exec test-postgres psql -U test_user -d test_db -c "SELECT COUNT(*) FROM processed_data;"
```

3. **Plugin Loading Errors**

```zsh
# Verify plugin build
ls -l bin/plugins/

# Check plugin permissions
chmod 755 bin/plugins/*.so
```

---

# Profiling

Profiling helps monitor performance by generating CPU and memory usage data.

1. Enable profiling in config:

```yaml
app:
  profiling: true
```

2. Generate profiles:

```zsh
# CPU profile
go tool pprof -http=:8080 cpu.pprof

# Memory profile
go tool pprof -http=:8080 mem.pprof
```

## Monitoring

You can access containers' hostnames by adding them to your local /etc/hosts file, mapping them to 127.0.0.1.

```zsh
sudo echo "127.0.0.1 grafana ddpm node-exporter prometheus pipeline-manager redpanda" >> /etc/hosts
```

The application provides the following endpoints:

  - DDPM Health Check: http://ddpm:8080/health
  - Grafana Dashboards: http://grafana:3000/dashboards
  - Prometheus Targets health: http://prometheus:9090/targets

Additional details on containers and ports:

  - Benthos metrics: 4195
  - Redpanda metrics: 9644
  - Node Exporter: 9100
  - Grafana: 3000
  - Prometheus: 9090

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

**1. Configuration Errors**

```zsh
# Validate YAML syntax
yamllint config/app-config.yaml

# Check environment variables
echo $CONFIG_PATH
```

**2. Connection Issues**

```zsh
# Verify services
docker compose ps

# Check network
docker network ls
docker network inspect test-network
```

**3. Plugin Issues**

```zsh
# Rebuild plugins
make build-plugins

# Check plugin logs
docker logs test-pipeline-manager | grep "plugin"
```
---

# Contributing

Contributions are welcome!

  1. Fork repository
  2. Create feature branch
  3. Add tests
  4. Submit pull request

---

# License

Distributed Data Pipeline Manager is licensed under the MIT License. See the LICENSE file for details.
