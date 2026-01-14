.PHONY: build build-docker clean test run version docker-run docker-stop

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S_UTC')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go build flags
LDFLAGS := -s -w \
	-X 'pont/version.Version=$(VERSION)' \
	-X 'pont/version.BuildTime=$(BUILD_TIME)' \
	-X 'pont/version.GitCommit=$(GIT_COMMIT)'

# Build binary
build:
	@echo "Building pont $(VERSION)..."
	@echo "  Version:    $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Git Commit: $(GIT_COMMIT)"
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o pont .
	@echo "Build complete: ./pont"

# Build Docker image
build-docker:
	@echo "Building Docker image with version $(VERSION)..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t pont:$(VERSION) \
		-t pont:latest \
		.
	@echo "Docker image built: pont:$(VERSION)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f pont
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run locally
run: build
	@echo "Starting pont..."
	./pont

# Run with Docker Compose
docker-run:
	@echo "Starting pont with Docker Compose..."
	docker-compose up -d
	@echo "pont is running. Use 'make docker-stop' to stop."

# Stop Docker Compose
docker-stop:
	@echo "Stopping pont..."
	docker-compose down

# Show version info
version:
	@echo "Version:    $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
