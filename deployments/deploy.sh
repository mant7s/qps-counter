#!/bin/bash

set -eo pipefail

# Validate dependencies
check_dependency() {
    if ! command -v $1 &> /dev/null; then
        echo "ERROR: $1 is required but not installed"
        exit 1
    fi
}

check_dependency docker
check_dependency docker-compose

# Cleanup old containers
echo "Cleaning up old containers..."
docker-compose down --remove-orphans || true

# Build and test
echo "Building application..."
make test || { echo "Unit tests failed"; exit 1; }
docker-compose build || { echo "Docker build failed"; exit 1; }

# Deploy
echo "Starting services..."
docker-compose up -d --scale qps-counter=3

# Health check
echo "Checking service health..."
for i in {1..10}; do
    if curl -s http://localhost:8080/healthz | grep -q "OK"; then
        echo "Service is healthy"
        break
    else
        if [ $i -eq 10 ]; then
            echo "ERROR: Service failed to start"
            docker-compose logs
            exit 1
        fi
        sleep 5
    fi
done

echo ""
echo "Deployment successful!"
echo "Access endpoints:"
echo "  - QPS API: http://localhost:8080/qps"
echo "  - Load Balancer: http://localhost"