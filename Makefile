BINARY_NAME=zopen-mcp-server
VERSION=1.1.0

.PHONY: all build run clean

all: build

build:
	@echo "Building the application v$(VERSION)..."
	@go build -o $(BINARY_NAME) zopen-server.go

run: build
	@echo "Running the application..."
	@./$(BINARY_NAME)

clean:
	@echo "Cleaning up..."
	@go clean
	@rm -f $(BINARY_NAME)

test:
	@echo "Running tests..."
	@./test.sh
