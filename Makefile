.PHONY: setup run test test-race lint migrate help

help:
	@echo "Available commands:"
	@echo "  make setup     - Download Go dependencies and tools"
	@echo "  make run       - Run the sandbox application"
	@echo "  make test      - Run unit and integration tests"
	@echo "  make test-race - Run tests with race condition detector"
	@echo "  make lint      - Run golangci-lint or go vet"

setup:
	go mod download
	go mod tidy

run:
	go run ./cmd/sandbox

test:
	go test -v ./...

test-race:
	go test -race -v ./...

lint:
	go vet ./...
