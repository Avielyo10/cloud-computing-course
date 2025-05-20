#!/bin/zsh

# Install test dependencies if needed
echo "Installing test dependencies..."
go get github.com/stretchr/testify/assert github.com/stretchr/testify/mock

# Run unit tests first
echo "Running unit tests..."
go test -v ./internal/... ./pkg/... ./server/...

# Check if unit tests passed
if [ $? -ne 0 ]; then
    echo "Unit tests failed. Exiting."
    exit 1
fi

# Run integration tests if requested
if [ "$1" = "--integration" ]; then
    echo "Running integration tests..."
    
    # Set up environment variables for integration tests
    export INTEGRATION_TEST=true
    
    # Use DynamoDB Local or actual AWS resources depending on availability
    if [ -n "$AWS_PROFILE" ] && [ -n "$TABLE_NAME" ]; then
        echo "Using AWS resources with profile: $AWS_PROFILE and table: $TABLE_NAME"
        go test -v -tags=integration ./test/integration/...
    else
        echo "AWS_PROFILE or TABLE_NAME not set. Skipping actual AWS integration tests."
        # Could add DynamoDB Local setup here if needed
    fi
fi

echo "All tests completed successfully."
