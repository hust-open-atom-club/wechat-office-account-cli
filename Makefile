GO ?= go
BINARY := weoa-cli
COMMAND := .
BIN_DIR := bin

GOOS ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)

EXE :=
ifeq ($(GOOS),windows)
EXE := .exe
endif

BIN := $(BIN_DIR)/$(BINARY)$(EXE)
COVERAGE_FILE ?= coverage.out

.DEFAULT_GOAL := build

.PHONY: all build install uninstall test test-race vet lint fmt fmt-check coverage clean help

all: lint test build

build:
	@mkdir -p $(BIN_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -trimpath -o $(BIN) $(COMMAND)

install:
	$(GO) install $(COMMAND)

uninstall:
	@rm -f "$$($(GO) env GOPATH)/bin/$(BINARY)$(EXE)"

test:
	$(GO) test ./... -count=1

test-race:
	$(GO) test -race ./... -count=1

vet:
	$(GO) vet ./...

lint: fmt-check vet

fmt:
	$(GO) fmt ./...

fmt-check:
	@test -z "$$(gofmt -l .)" || { \
		echo "The following files need formatting:"; \
		gofmt -l .; \
		exit 1; \
	}

coverage:
	$(GO) test ./... -count=1 -coverprofile=$(COVERAGE_FILE)
	$(GO) tool cover -func=$(COVERAGE_FILE)

clean:
	@rm -rf $(BIN_DIR)
	@rm -f $(COVERAGE_FILE)

help:
	@echo "weoa-cli development targets:"
	@echo ""
	@echo "Build and installation:"
	@echo "  make build                  Build $(BIN) for GOOS=$(GOOS) GOARCH=$(GOARCH)"
	@echo "  make build GOOS=linux       Cross-compile for Linux"
	@echo "  make install                Install to GOPATH/bin with go install"
	@echo "  make uninstall              Remove the binary from GOPATH/bin"
	@echo ""
	@echo "Checks:"
	@echo "  make test                   Run the standard test suite"
	@echo "  make test-race              Run tests with race detection"
	@echo "  make lint                   Check formatting and run go vet"
	@echo "  make coverage               Generate $(COVERAGE_FILE) and print coverage"
	@echo ""
	@echo "Maintenance:"
	@echo "  make fmt                    Format Go source files in place"
	@echo "  make clean                  Remove build and coverage files"
