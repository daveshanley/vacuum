.PHONY: help build build-ui test run

.DEFAULT_GOAL := help

help: ## Display this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build-ui: ## Build HTML report UI assets
	@./scripts/build-ui-assets.sh

build: build-ui ## Build the vacuum binary to bin/vacuum
	@go build -tags html_report_ui -o bin/vacuum

test: build-ui ## Run the Go test suite
	@go test ./...
	@go test -tags html_report_ui ./...

run: ## Run vacuum directly with go run
	@go run vacuum.go
