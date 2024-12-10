#!/bin/bash

run_tests() {
    local format=$1
    
    # Validate format directory exists
    if [ ! -d "tests/integration/plugins/$format" ]; then
        echo "Error: Plugin directory for format '$format' does not exist"
        return 1
    fi

    echo "Starting test environment..."
    docker compose --profile testing -f tests/integration/docker-compose.test.yml up -d

    echo "Running tests..."
    docker exec test-pipeline-manager go test ./tests/integration/plugins/$format/... -v
    local test_exit_code=$?

    echo "Cleaning up..."
    docker compose --profile testing -f tests/integration/docker-compose.test.yml down -v

    return $test_exit_code
}

format=$1
run_tests $format
exit $?
