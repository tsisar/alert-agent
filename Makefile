# ==============================
# Build configuration
# ==============================

TARGETOS     ?= linux
TARGETARCH   ?= amd64
CGO_ENABLED  ?= 0

APP        ?= alert-agent
REGISTRY   ?= ghcr.io/tsisar
VERSION    := $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)-$(shell git rev-parse --short HEAD)
COMMIT     := $(shell git rev-parse --short HEAD)
DATE       := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -X github.com/tsisar/alert-agent/internal/version.Version=$(VERSION) \
              -X github.com/tsisar/alert-agent/internal/version.Commit=$(COMMIT) \
              -X github.com/tsisar/alert-agent/internal/version.Date=$(DATE)

# ==============================
# Help
# ==============================
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  help     - Show this help message"
	@echo "  format   - Format Go code"
	@echo "  lint     - Run golangci-lint"
	@echo "  test     - Run tests"
	@echo "  generate - Generate templ code"
	@echo "  get      - Get dependencies"
	@echo "  run      - Run the application locally"
	@echo "  build    - Build the production binary"
	@echo "  dev      - Build the development binary"
	@echo "  image    - Build Docker image"
	@echo "  push     - Push Docker image to registry"
	@echo "  test-webhook - Send test Grafana alert to localhost:8080"
	@echo "  clean    - Clean build artifacts"
	@echo "  build-and-push - clean -> test -> build -> image -> push"
	@echo "  release  - Create and push a release tag from the release branch (usage: make release VERSION=v1.0.0)"
	@echo ""
	@echo "Configuration:"
	@echo "  TARGETOS     = $(TARGETOS)"
	@echo "  TARGETARCH   = $(TARGETARCH)"
	@echo "  CGO_ENABLED  = $(CGO_ENABLED)"
	@echo "  VERSION      = $(VERSION)"

# ==============================
# Go Tools
# ==============================
.PHONY: format
format:
	@echo "Formatting Go code..."
	@gofmt -s -w ./

.PHONY: install-lint
install-lint:
	@which golangci-lint >/dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)

.PHONY: lint
lint: install-lint
	@echo "Running linter..."
	@golangci-lint run ./...

.PHONY: test
test:
	@echo "Running tests..."
	@go test -v -cover ./...

.PHONY: install-templ
install-templ:
	@which templ >/dev/null || (echo "Installing templ..." && go install github.com/a-h/templ/cmd/templ@latest)

.PHONY: generate
generate: install-templ
	@echo "Generating templ code..."
	@templ generate ./internal/web/templates/

.PHONY: get
get:
	@echo "Getting dependencies..."
	@go mod tidy
	@go mod download

.PHONY: run
run: generate
	@echo "Running the application..."
	@CGO_ENABLED=$(CGO_ENABLED) go run ./cmd/alert-agent serve

# ==============================
# Build targets
# ==============================
.PHONY: dev
dev: format generate get
	@echo "Building development binary..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) \
		go build -v -ldflags="$(LDFLAGS)" -o bin/$(APP) ./cmd/alert-agent

.PHONY: build
build: format generate get
	@echo "Building production binary..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) \
		go build -v -ldflags="-s -w $(LDFLAGS)" -o bin/$(APP)-$(TARGETOS)-$(TARGETARCH) ./cmd/alert-agent

# ==============================
# Docker
# ==============================
.PHONY: image
image:
	@echo "Building Docker image for $(TARGETOS)/$(TARGETARCH)..."
	@docker buildx build \
		--platform $(TARGETOS)/$(TARGETARCH) \
		--build-arg TARGETOS=$(TARGETOS) \
		--build-arg TARGETARCH=$(TARGETARCH) \
		--tag $(REGISTRY)/$(APP):$(VERSION) \
		--load .

.PHONY: push
push:
	@echo "Pushing image to registry..."
	@if [ -n "$$CI_REGISTRY_USER" ] && [ -n "$$CI_REGISTRY_PASSWORD" ]; then \
		echo "$$CI_REGISTRY_PASSWORD" | docker login -u "$$CI_REGISTRY_USER" --password-stdin "$$CI_REGISTRY"; \
	fi; \
	docker push $(REGISTRY)/$(APP):$(VERSION)

.PHONY: print-image
print-image:
	@echo $(REGISTRY)/$(APP):$(VERSION)

# ==============================
# Testing
# ==============================
.PHONY: test-webhook
test-webhook:
	@echo "Sending test webhook to localhost:8080..."
	@curl -s -X POST http://localhost:8080/webhook \
		-H "Content-Type: application/json" \
		-d '{ \
			"version": "1", \
			"status": "firing", \
			"receiver": "alert-agent", \
			"externalURL": "http://grafana.example.com/", \
			"groupKey": "{}/{alertname=TestAlert}", \
			"groupLabels": {"alertname": "TestAlert"}, \
			"commonLabels": {"alertname": "TestAlert", "severity": "warning"}, \
			"commonAnnotations": {"summary": "Synthetic alert from make test-webhook"}, \
			"alerts": [{ \
				"status": "firing", \
				"labels": {"alertname": "TestAlert", "severity": "warning"}, \
				"annotations": {"summary": "Synthetic alert from make test-webhook", "description": "This payload is generated inline by the Makefile; no test fixtures required."}, \
				"startsAt": "2026-01-01T00:00:00Z", \
				"endsAt": "0001-01-01T00:00:00Z", \
				"generatorURL": "http://grafana.example.com/", \
				"fingerprint": "test-webhook-fingerprint" \
			}] \
		}' | jq .

# ==============================
# Utilities
# ==============================
.PHONY: clean
clean:
	@echo "Cleaning artifacts..."
	@rm -rf bin/
	@docker rmi $(REGISTRY)/$(APP):$(VERSION) 2>/dev/null || true

.PHONY: build-and-push
build-and-push: clean test build image push
	@echo "Release $(VERSION) completed successfully"

# ==============================
# Release tagging
# ==============================
# Usage: make release VERSION=v1.0.0
#
# Switches to the `release` branch, pulls, then creates and pushes an
# annotated semver tag. CI then builds the image, pushes :vX.Y.Z
# (+ :latest for stable tags), and creates the GitLab Release.
.PHONY: release
release:
	@git checkout release
	@git pull
	@if [ -z "$(VERSION)" ]; then \
		echo "ERROR: VERSION is required. Usage: make release VERSION=v1.0.0"; exit 1; \
	fi
	@echo "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$$' || \
		(echo "ERROR: VERSION must match vX.Y.Z[-suffix], got: $(VERSION)"; exit 1)
	@branch=$$(git rev-parse --abbrev-ref HEAD); \
		if [ "$$branch" != "release" ]; then \
			echo "ERROR: tags must be created on the 'release' branch (current: $$branch)"; exit 1; \
		fi
	@if ! git diff-index --quiet HEAD --; then \
		echo "ERROR: working tree has uncommitted changes"; exit 1; \
	fi
	@if git rev-parse "$(VERSION)" >/dev/null 2>&1; then \
		echo "ERROR: tag $(VERSION) already exists"; exit 1; \
	fi
	@git fetch origin --tags --quiet
	@if [ "$$(git rev-parse HEAD)" != "$$(git rev-parse origin/release)" ]; then \
		echo "ERROR: local 'release' is not in sync with origin/release (run: git pull)"; exit 1; \
	fi
	@echo "Creating annotated tag $(VERSION)..."
	@git tag -a "$(VERSION)" -m "Release $(VERSION)"
	@git push origin "$(VERSION)"
	@echo "Tag $(VERSION) pushed. CI will build the image and create the GitLab Release."
