.PHONY: default build test lint format format-check generate clean clean-coverage ensure-gotestsum

export GO111MODULE=on

GOTESTSUM ?= gotestsum
TEST_FORMAT ?= pkgname-and-test-fails
COVER_DIR := .out/cover
JUNIT_FILE := .out/junit.xml
LCOV_FILE := .out/lcov.info

default: test

build:
	go build ./...

test: clean-coverage ensure-gotestsum
	mkdir -p $(dir $(LCOV_FILE))
	$(GOTESTSUM) --junitfile $(JUNIT_FILE) --format $(TEST_FORMAT) -- -coverpkg=./... -covermode=atomic -coverprofile=$(LCOV_FILE) -count=1 ./...

lint:
	golangci-lint run --verbose

format:
	dprint fmt
	go fmt ./...

format-check:
	dprint check
	test -z "$$(gofmt -l .)"

generate:
	go generate ./...

clean-coverage:
	rm -rf .out

clean: clean-coverage
	rm -rf build

ensure-gotestsum:
	@if ! command -v $(GOTESTSUM) >/dev/null 2>&1; then \
		GOFLAGS=-mod=mod go install gotest.tools/gotestsum@latest; \
	fi
