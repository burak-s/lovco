.PHONY: build run clean docker-build docker-up docker-down docker-logs docker-run-env dev-up help

BINARY_NAME=lovco
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin

build:
	go build -o $(GOBIN)/$(BINARY_NAME) server/cmd/main.go

run:
	@echo "Running $(BINARY_NAME) on port 50051"
	go run server/cmd/main.go -port 50051 -address 0.0.0.0

clean:
	@echo "Cleaning up $(BINARY_NAME)"
	@rm -rf $(GOBIN)/$(BINARY_NAME)
	@rm -rf $(GOBASE)/server/chat/*.pb.go
	@rm -rf $(GOBASE)/server/leftover/*.pb.go
	@go clean

help:
	@echo "Usage: make <target>"
	@echo "Targets:"
	@echo "  build: Build the $(BINARY_NAME) binary"
	@echo "  run: Run the the application"
	@echo "  clean: Clean up the application"
	@echo "  docker-run-env: Run Docker container with environment variables"
	@echo "  help: Show this help message"
