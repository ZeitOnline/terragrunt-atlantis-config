.PHONY: test
test: ## Run tests
	mkdir -p cmd/test_artifacts
	go test -v ./...
	rm -rf cmd/test_artifacts

.PHONY: build
build: test ## Build for current platform
	goreleaser build --snapshot --clean --single-target

.PHONY: build-all
build-all: test ## Build for all platforms
	goreleaser build --snapshot --clean

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf ./dist ./build
	go clean

.PHONY: release
release: ## Create a release (requires proper git tag)
	goreleaser release --clean

.PHONY: help
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
