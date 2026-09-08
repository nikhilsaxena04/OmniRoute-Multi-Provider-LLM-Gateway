.PHONY: build run test docker-up docker-down clean

# Build the Go gateway
build:
	cd gateway && go build -o omni-router main.go

# Run the gateway locally
run: build
	cd gateway && ./omni-router

# Run all unit tests
test:
	cd gateway && go test ./... -v

# Spin up the entire stack (Gateway, Prometheus, Grafana) using Docker Compose
docker-up:
	docker-compose up -d --build

# Tear down the stack
docker-down:
	docker-compose down

# Clean binaries
clean:
	rm -f gateway/omni-router
