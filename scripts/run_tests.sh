#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

# Function to display usage
usage() {
    echo "Usage: $0 [--unit|--integration|--all]"
    echo ""
    echo "Runs the specified tests for the parking lot application."
    echo ""
    echo "Options:"
    echo "  --unit         Run only unit tests."
    echo "  --integration  Run only integration tests (requires Docker and AWS CLI)."
    echo "  --all          Run all tests (unit and integration)."
    echo "  -h, --help     Display this help message."
    echo ""
    echo "Default behavior (no options): Run unit tests, then integration tests if Docker and AWS CLI are available."
}

# --- Dependency Checks ---
check_docker() {
    if ! command -v docker &> /dev/null; then
        echo "Error: Docker is not installed or not in PATH. Please install Docker to run integration tests."
        return 1
    fi
    if ! docker info &> /dev/null; then
        echo "Error: Docker daemon is not running. Please start Docker to run integration tests."
        return 1
    fi
    return 0
}

check_aws_cli() {
    if ! command -v aws &> /dev/null; then
        echo "Error: AWS CLI is not installed or not in PATH. Please install AWS CLI to run integration tests."
        echo "See: https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html"
        return 1
    fi
    return 0
}
# --- End Dependency Checks ---

# Variables
DYNAMODB_CONTAINER_NAME="dynamodb-local-test"
TABLE_NAME="parking_tickets"
AWS_ENDPOINT_URL="http://localhost:8000"
AWS_REGION="us-east-1" # Required by AWS CLI, even for local
AWS_ACCESS_KEY_ID="dummy" # Required by AWS CLI, even for local
AWS_SECRET_ACCESS_KEY="dummy" # Required by AWS CLI, even for local

# Default test types
RUN_UNIT_TESTS=false
RUN_INTEGRATION_TESTS=false

# Parse command-line arguments
if [ $# -eq 0 ]; then
    # Default behavior: try to run both if dependencies met
    RUN_UNIT_TESTS=true
    if check_docker && check_aws_cli; then
        RUN_INTEGRATION_TESTS=true
    else
        echo "Skipping integration tests due to missing dependencies (Docker or AWS CLI)."
    fi
else
    while [[ "$1" != "" ]]; do
        case $1 in
            --unit )
                RUN_UNIT_TESTS=true
                ;;
            --integration )
                if check_docker && check_aws_cli; then
                    RUN_INTEGRATION_TESTS=true
                else
                    echo "Error: Cannot run integration tests. Docker and/or AWS CLI are not available."
                    exit 1
                fi
                ;;
            --all )
                RUN_UNIT_TESTS=true
                if check_docker && check_aws_cli; then
                    RUN_INTEGRATION_TESTS=true
                else
                    echo "Error: Cannot run all tests. Docker and/or AWS CLI are not available for integration tests."
                    exit 1
                fi
                ;;
            -h | --help )
                usage
                exit 0
                ;;
            * )
                usage
                exit 1
        esac
        shift
    done
fi

if ! $RUN_UNIT_TESTS && ! $RUN_INTEGRATION_TESTS; then
    echo "No tests selected to run. Exiting."
    usage
    exit 0
fi


# Function to clean up DynamoDB container
cleanup() {
    echo "Cleaning up DynamoDB container..."
    if docker ps -a --format '{{.Names}}' | grep -q "^${DYNAMODB_CONTAINER_NAME}$"; then
        docker rm -f $DYNAMODB_CONTAINER_NAME
        echo "DynamoDB container removed."
    else
        echo "DynamoDB container not found or already removed."
    fi
}

# Trap EXIT signal to ensure cleanup
trap cleanup EXIT

if $RUN_UNIT_TESTS; then
    echo "Installing test dependencies..."
    go get -t ./...
    echo "Running unit tests..."
    go test -v -coverprofile=coverage-unit.out ./internal/... ./pkg/... ./server/...
    if [ -f coverage-unit.out ]; then
        go tool cover -func=coverage-unit.out
    fi
fi

if $RUN_INTEGRATION_TESTS; then
    echo "Running integration tests..."

    # Export AWS credentials and endpoint configuration for AWS CLI commands
    export AWS_ENDPOINT_URL
    export AWS_REGION
    export AWS_ACCESS_KEY_ID
    export AWS_SECRET_ACCESS_KEY
    export DYNAMODB_TABLE_NAME=$TABLE_NAME # Also export table name for Go tests

    # Ensure no previous container is running
    if docker ps -a --format '{{.Names}}' | grep -q "^${DYNAMODB_CONTAINER_NAME}$"; then
        echo "Found existing DynamoDB container. Removing it..."
        docker rm -f $DYNAMODB_CONTAINER_NAME
    fi

    echo "Starting DynamoDB Local in Docker..."
    # Removed -dbPath /data as it was causing issues and is not strictly necessary for ephemeral testing
    docker run -d -p 8000:8000 --name $DYNAMODB_CONTAINER_NAME amazon/dynamodb-local

    # Check if the container is running
    if ! docker ps -f name=$DYNAMODB_CONTAINER_NAME --format '{{.Names}}' | grep -q "^${DYNAMODB_CONTAINER_NAME}$"; then
        echo "Error: Docker container $DYNAMODB_CONTAINER_NAME failed to start."
        echo "Attempting to get logs from container (if it exists):"
        docker logs $DYNAMODB_CONTAINER_NAME || echo "Could not retrieve logs for $DYNAMODB_CONTAINER_NAME."
        exit 1 # Exit if container didn't start
    fi

    # Wait for DynamoDB Local to be ready
    echo "Waiting for DynamoDB Local to start..."
    retries=0
    max_retries=30 # Wait for max 30 seconds
    health_check_passed=false
    # Ensure AWS CLI output is not paginated and errors are visible
    # Loop until health_check_passed is true or max_retries is reached
    until [ "$health_check_passed" = true ] || [ $retries -eq $max_retries ]; do
        aws_output=$(aws dynamodb list-tables --endpoint-url ${AWS_ENDPOINT_URL} --region ${AWS_REGION} --no-cli-pager 2>&1)
        aws_exit_code=$?
        
        if [ $aws_exit_code -eq 0 ]; then
            health_check_passed=true
        fi
        
        if [ "$health_check_passed" = false ]; then
            retries=$((retries+1))
            if [ $retries -lt $max_retries ]; then # Avoid sleeping if it's the last attempt and will exit loop
                sleep 1
            fi
        fi
    done

    if [ "$health_check_passed" = false ]; then
        echo "DynamoDB Local did not start in time or health check consistently failed after $max_retries attempts."
        # The variables $aws_exit_code and $aws_output will hold the values from the last attempt
        echo "Final attempt's AWS CLI exit code: $aws_exit_code"
        echo "Final attempt's AWS CLI output: $aws_output"
        echo "Displaying full DynamoDB container logs:"
        docker logs $DYNAMODB_CONTAINER_NAME
        echo "Removing DynamoDB container..."
        docker rm -f $DYNAMODB_CONTAINER_NAME
        exit 1
    fi
    echo "DynamoDB Local started and health check passed."

    # Create DynamoDB table for testing
    echo "Creating DynamoDB table: $TABLE_NAME..."
    aws dynamodb create-table \
        --table-name $TABLE_NAME \
        --attribute-definitions AttributeName=ticket_id,AttributeType=S \
        --key-schema AttributeName=ticket_id,KeyType=HASH \
        --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
        --endpoint-url ${AWS_ENDPOINT_URL} \
        --region ${AWS_REGION} > /dev/null # Suppress verbose output

    echo "Table $TABLE_NAME created."

    echo "Running integration tests with DynamoDB Local..."
    # Run integration tests specifically, including the 'integration' build tag
    go test -v -tags=integration -coverprofile=coverage-integration.out ./test/integration
    if [ -f coverage-integration.out ]; then
        go tool cover -func=coverage-integration.out
    fi

    # Unset environment variables
    unset AWS_ENDPOINT_URL
    unset AWS_REGION
    unset AWS_ACCESS_KEY_ID
    unset AWS_SECRET_ACCESS_KEY
    unset DYNAMODB_TABLE_NAME

    # Cleanup is handled by the trap
fi

echo "Test run finished."
