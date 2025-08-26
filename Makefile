BINARY_NAME=zopen-mcp-server

.PHONY: all build run clean

all: build

build:
	@echo "Building the application..."
	@go build -o $(BINARY_NAME) zopen-server.go

run: build
	@echo "Running the application..."
	@./$(BINARY_NAME)

clean:
	@echo "Cleaning up..."
	@go clean
	@rm -f $(BINARY_NAME)
