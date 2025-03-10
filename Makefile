.PHONY: all build test lint run clean docker-build docker-run benchmark

GO_SOURCES := $(shell find . -type f -name '*.go')
VERSION := $(shell git describe --tags 2>/dev/null || echo "v0.0.1")

all: build

build: $(GO_SOURCES)
	@echo "Building application..."
	@go build -ldflags "-X main.version=$(VERSION)" -o bin/qps-counter ./cmd/server

test:
	@echo "Running tests..."
	@go test -race -coverprofile=coverage.out -covermode=atomic ./...

lint:
	@echo "Running linters..."
	@golangci-lint run

run: build
	@echo "Starting service..."
	@./bin/qps-counter -config=./config/config.yaml

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/ coverage.out

docker-build:
	@echo "Building Docker image..."
	@docker-compose build

docker-run:
	@echo "Starting services..."
	@docker-compose up -d

benchmark:
	@wrk -t4 -c100 -d30s --latency http://localhost:8080

migrate-config:
	@echo "Updating config schema..."
	@cp config/config.yaml config/config.yaml.bak
	@jq --arg version $(VERSION) '.version = $$version' config/config.yaml > tmp && mv tmp config/config.yaml

release: test lint
	@echo "Creating release $(VERSION)"
	@git tag $(VERSION)
	@git push origin $(VERSION)