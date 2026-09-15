# task — a local task list
#
# Run "make" or "make help" to see the available targets.

BINARY := task
CMD    := ./cmd/task
BIN    := bin
GOBIN  ?= $(shell go env GOPATH)/bin

# The version is asked of git rather than written down: a constant in the source
# is a thing to forget, and a forgotten one lies. On the tag this is "v1.0.0",
# past it "v1.0.0-3-gabc1234", and with uncommitted changes it says "-dirty".
VERSION := $(shell git describe --tags --dirty --always 2>/dev/null || echo devel)
LDFLAGS := -X main.version=$(VERSION)

# Arguments forwarded by "make run", e.g. make run ARGS="ls -a"
ARGS ?=

.DEFAULT_GOAL := help

.PHONY: help build install run test cover fmt vet tidy check clean

help: ## Show this help
	@grep -hE '^[a-z-]+:.*## ' $(MAKEFILE_LIST) \
		| awk -F':.*## ' '{printf "  \033[1m%-8s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary into bin/
	@mkdir -p $(BIN)
	go build -ldflags "$(LDFLAGS)" -o $(BIN)/$(BINARY) $(CMD)

install: ## Install the binary into GOBIN
	go install -ldflags "$(LDFLAGS)" $(CMD)

run: ## Run without installing, e.g. make run ARGS="ls -a"
	go run $(CMD) $(ARGS)

test: ## Run the test suite
	go test ./...

cover: ## Report test coverage and open it in a browser
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

fmt: ## Rewrite source files with gofmt
	gofmt -w .

vet: ## Run go vet
	go vet ./...

tidy: ## Tidy go.mod and go.sum
	go mod tidy

check: ## Verify formatting, vet, and tests — run this before committing
	@unformatted=$$(gofmt -l .); \
		if [ -n "$$unformatted" ]; then \
			echo "gofmt needed:"; echo "$$unformatted"; exit 1; \
		fi
	go vet ./...
	go test ./...

clean: ## Remove build output and coverage data
	rm -rf $(BIN) coverage.out
