GOCMD=go
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOINSTALL =$(GOCMD) install
GOMOD=$(GOCMD) mod
GOGENERATE=$(GOCMD) generate

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: all test coverage deps generate lint fmt vet check examples help

all: deps generate fmt vet lint test ## Run all development tasks

help: ## Display this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run tests (excluding examples)
	$(GOTEST) -v -race $(shell go list ./... | grep -v examples)

test-verbose: ## Run tests with verbose output
	$(GOTEST) -v -race ./...

test-short: ## Run short tests only
	$(GOTEST) -short $(shell go list ./... | grep -v examples)

coverage: ## Run tests with coverage
	$(GOTEST) -race -coverprofile=coverage.out -covermode=atomic $(shell go list ./... | grep -v examples)
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean-coverage: ## Clean coverage files
	rm -f coverage.out coverage.html

coverage-func: ## Show coverage by function
	$(GOTEST) -race -coverprofile=coverage.out -covermode=atomic $(shell go list ./... | grep -v examples)
	$(GOCMD) tool cover -func=coverage.out

examples: ## Run example tests
	$(GOTEST) -v ./examples/...

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

deps-update: ## Update dependencies
	$(GOGET) -u ./...
	$(GOMOD) tidy

generate: ## Run go generate
	$(GOGENERATE) ./...

lint: ## Run golangci-lint
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

fmt: ## Format code
	@find . -name "*.go" -not -name "*_easyjson.go"  | xargs gofmt -s -w
	@find . -name "*.go" -not -name "*_easyjson.go"  | xargs goimports -w
	@find . -name "*.go" -not -name "*_easyjson.go"  | xargs golines -w

vet: ## Run go vet
	$(GOCMD) vet ./...

check: fmt vet lint ## Run all checks (format, vet, lint)

mod-verify: ## Verify dependencies
	$(GOMOD) verify

install-tools: ## Install development tools
	$(GOINSTALL) github.com/mailru/easyjson/easyjson@latest
	$(GOINSTALL) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOINSTALL) golang.org/x/tools/cmd/goimports@latest
	$(GOINSTALL) github.com/segmentio/golines@latest
