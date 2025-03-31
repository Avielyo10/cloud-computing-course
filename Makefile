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

clean:
	@echo "Cleaning build artifacts..."
	rm -f cmd/lambda/*.zip
	@echo "Cleaned."