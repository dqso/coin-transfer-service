.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: generate
generate: ## Generate code
	go generate ./...

.PHONY: test
test: ## Run unit tests
	go test -race ./...

.PHONY: test-integration
test-integration: ## Run integration tests (needs Docker)
	go test -race -tags integration -count=1 ./app/tests/integration/

.PHONY: test-integration-verbose
test-integration-verbose: ## Run integration tests one by one with request logs
	go test -race -tags integration -count=1 -v -parallel 1 ./app/tests/integration/