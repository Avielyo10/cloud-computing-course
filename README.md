# Parking Lot Management System

This project is a c## Project Structure

```shell
├── cmd/lambda        # Lambda handler entry point
├── deployment        # Pulumi deployment code
├── internal
│   ├── handler       # API request handlers
│   ├── model         # Data models
│   └── service       # Business logic services
├── pkg
│   └── lambda        # Lambda adapter
├── server
│   └── api           # Generated API code
└── spec              # API specifications
```

## Testing

The project includes a comprehensive test suite with unit tests and integration tests.

### Running Tests

To run unit tests:

```bash
# Run all unit tests
make test

# Run tests with coverage report
make coverage
```

For manual test running:

```bash
# Run all unit tests
go test -v ./...

# Run specific package tests
go test -v ./internal/service

# Run with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Integration Tests

Integration tests require AWS credentials or DynamoDB Local:

```bash
# Run with integration tests (requires AWS credentials)
./scripts/run_tests.sh --integration
```

Set the following environment variables for integration tests:

- `AWS_PROFILE`: Your AWS profile with DynamoDB permissions
- `TABLE_NAME`: Name of a DynamoDB table to use for testing
- `INTEGRATION_TEST=true`: Enables integration test mode

### Continuous Integration

GitHub Actions workflows are included to run tests on every push and pull request.

## Getting Startedg lot management system built with Go and AWS serverless technologies. The system provides API endpoints for managing vehicle entry and exit from parking lots, with automatic ticket generation and fee calculation based on parking duration.

## Architecture

The system is built with a serverless architecture using:

- **Frontend**: REST API (API Gateway)
- **Backend**: Go with Gin framework running on AWS Lambda
- **Database**: Amazon DynamoDB
- **Infrastructure**: Defined as code using Pulumi

### System Components

```
┌─────────────┐     ┌──────────────┐     ┌───────────────┐
│  API Gateway │────▶│  AWS Lambda  │────▶│    DynamoDB   │
└─────────────┘     └──────────────┘     └───────────────┘
       ▲                   │                     │
       │                   │                     │
       └───────────────────┴─────────────────────┘
```

## Features

- **Vehicle Entry**: Records vehicle entry with license plate and assigns unique ticket ID
- **Vehicle Exit**: Processes vehicle exit with fee calculation
- **Parking Fee Calculation**: Based on parking duration ($0.10/minute with $5 minimum)
- **Serverless Operation**: Scales automatically with demand
- **Cloud-Native**: Designed for AWS cloud environment

## API Endpoints

### Record Vehicle Entry

```
POST /entry?plate={licensePlate}&parkingLot={lotID}
```

- Records vehicle entry and generates a ticket
- Returns a ticket ID for future reference

### Process Vehicle Exit

```
POST /exit?ticketId={ticketID}
```

- Processes vehicle exit
- Returns details including license plate, parking lot, duration, and charge

## Project Structure

```
├── cmd/lambda        # Lambda handler entry point
├── deployment        # Pulumi deployment code
├── internal
│   ├── handler       # API request handlers
│   ├── model         # Data models
│   └── service       # Business logic services
├── pkg
│   └── lambda        # Lambda adapter
├── server
│   └── api           # Generated API code
└── spec              # API specifications
```

## Testing

The project includes a comprehensive test suite with unit tests and integration tests.

### Running Tests

To run unit tests:

```bash
# Run all unit tests
make test

# Run tests with coverage report
make coverage
```

For manual test running:

```bash
# Run all unit tests
go test -v ./...

# Run specific package tests
go test -v ./internal/service

# Run with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Integration Tests

Integration tests require AWS credentials or DynamoDB Local:

```bash
# Run with integration tests (requires AWS credentials)
./scripts/run_tests.sh --integration
```

Set the following environment variables for integration tests:

- `AWS_PROFILE`: Your AWS profile with DynamoDB permissions
- `TABLE_NAME`: Name of a DynamoDB table to use for testing
- `INTEGRATION_TEST=true`: Enables integration test mode

### Continuous Integration

GitHub Actions workflows are included to run tests on every push and pull request.

## Getting Started

### Prerequisites

- Go 1.24+
- AWS CLI configured
- Pulumi CLI
- Make

### Setup

1. Clone the repository:

   ```
   git clone https://github.com/Avielyo10/parking-lot.git
   cd parking-lot
   ```

2. Install dependencies:

   ```
   go mod download
   ```

### Local Development

To build the Lambda functions:

```
make build
```

### Deployment

1. Deploy infrastructure with Pulumi:

   ```
   cd deployment
   pulumi up
   ```

2. Pulumi will provision:
   - DynamoDB table for storing parking tickets
   - Lambda functions for handling entry and exit
   - API Gateway for exposing the endpoints
   - IAM roles for proper permissions

## Development Workflow

1. Make changes to the API specification in openapi.yaml
2. Run `make generate` to regenerate API code
3. Implement handlers in handler
4. Build Lambda functions using `make build`
5. Deploy with `pulumi up`

## License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.
