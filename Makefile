.PHONY: test build install

BIN ?= diffmate
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS ?= -X github.com/imadys/diffmate/internal/version.Version=$(VERSION)

test:
	go test ./...

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BIN) ./cmd/diffmate

install:
	go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/$(BIN) ./cmd/diffmate
