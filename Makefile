.PHONY: test build install

BIN ?= diffmate
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin

test:
	go test ./...

build:
	go build -o bin/$(BIN) ./cmd/diffmate

install:
	go build -o $(BINDIR)/$(BIN) ./cmd/diffmate
