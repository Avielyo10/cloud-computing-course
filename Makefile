generate:
	@echo "Generating code..."
	oapi-codegen -config spec/gin-server.yaml ./spec/openapi.yaml

fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "Code formatted."

build: generate fmt
	@echo "Building Lambda handler..."
	cd cmd/lambda && \
	GOOS=linux GOARCH=arm64 go build -o bootstrap main.go && \
	zip entry-handler.zip bootstrap && \
	zip exit-handler.zip bootstrap && \
	rm -f bootstrap
	@echo "Lambda handler built."

test:
	@echo "Running tests..."
	go test -v ./...
	@echo "Tests completed."

coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated in coverage.html"

clean:
	@echo "Cleaning build artifacts..."
	rm -f cmd/lambda/*.zip
	rm -f coverage.out coverage.html
	@echo "Cleaned."