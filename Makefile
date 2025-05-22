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
	rm -f bootstrap entry-handler.zip exit-handler.zip && \
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

deploy: clean test build
	@if [ -z "$$AWS_ACCESS_KEY_ID" ] || [ -z "$$AWS_SECRET_ACCESS_KEY" ] || [ -z "$$AWS_REGION" ]; then \
		echo "Error: AWS credentials not set. Please set AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY and AWS_REGION environment variables."; \
		exit 1; \
	fi
	@echo "Deploying infrastructure using Terraform..."
	cd deployment && \
	terraform init && \
	terraform plan -out=tfplan && \
	TF_LOG=DEBUG terraform apply -auto-approve tfplan
	@echo "Deployment completed."
