.PHONY: build test lint clean fmt tidy ci

GO ?= go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.0.0-dev")
LDFLAGS ?= -s -w
BINARY ?= ai-runtime

build:
	$(GO) build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/module

test:
	$(GO) test -race -count=1 -timeout 60s ./...

lint:
	golangci-lint run --timeout 120s ./...

clean:
	rm -f $(BINARY)

fmt:
	$(GO) fmt ./...

tidy:
	$(GO) mod tidy

ci: lint test build
