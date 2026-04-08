APP_NAME := sh
GO := go
GOFLAGS := -v
LDFLAGS := -ldflags "-s -w"
BUILD_DIR := build
BINARY := $(BUILD_DIR)/$(APP_NAME)

SOURCES := $(wildcard *.go)

.PHONY: all
all: deps build

.PHONY: init
init:
	@if [ ! -f go.mod ]; then \
		$(GO) mod init $(APP_NAME); \
	fi

.PHONY: deps
deps: init
	$(GO) mod download
	$(GO) mod tidy

.PHONY: build
build: deps
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY) .

.PHONY: build-debug
build-debug: deps
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BINARY) .

.PHONY: run
run: build
	./$(BINARY)

.PHONY: run-fast
run-fast:
	$(GO) run .

.PHONY: test
test: deps
	$(GO) test -v ./...

.PHONY: test-coverage
test-coverage: deps
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

.PHONY: fmt
fmt:
	$(GO) fmt ./...
	gofmt -s -w .

.PHONY: lint
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

.PHONY: vet
vet: deps
	$(GO) vet ./...

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

.PHONY: install
install: deps
	$(GO) install .

.PHONY: build-linux
build-linux: deps
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 .

build-darwin: deps
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 .

.PHONY: build-windows
build-windows: deps
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe .

.PHONY: build-all
build-all: build-linux build-darwin build-windows

.PHONY: verify
verify: deps
	$(GO) mod verify

.PHONY: update-deps
update-deps:
	$(GO) get -u ./...
	$(GO) mod tidy

.PHONY: dev
dev:
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not installed. Run: go install github.com/cosmtrek/air@latest"; \
		$(GO) run .; \
	fi

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all            - Download deps and build (default)"
	@echo "  init           - Initialize go module"
	@echo "  deps           - Download dependencies"
	@echo "  build          - Build optimized binary"
	@echo "  build-debug    - Build without optimizations"
	@echo "  run            - Build and run"
	@echo "  run-fast       - Run directly (no binary)"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code (requires golangci-lint)"
	@echo "  vet            - Vet code"
	@echo "  clean          - Remove build artifacts"
	@echo "  install        - Install binary to GOPATH/bin"
	@echo "  build-all      - Cross-compile for all platforms"
	@echo "  build-linux    - Build for Linux (amd64/arm64)"
	@echo "  build-darwin   - Build for macOS (amd64/arm64)"
	@echo "  build-windows  - Build for Windows"
	@echo "  verify         - Verify dependencies"
	@echo "  update-deps    - Update all dependencies"
	@echo "  dev            - Development mode with hot reload"
	@echo "  help           - Show this help"
