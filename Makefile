MAKEFLAGS := --no-builtin-rules
SHELL := bash
.ONESHELL:

export CGO_ENABLED := 0

BINARY := bin/pidonetest

build: $(BINARY)

$(BINARY): $(wildcard ./cmd/pidonetest/*)
	go build -o $@ ./cmd/pidonetest/

.PHONY: clean
clean:
	rm -rfv bin

.PHONY: test
test: build
	go test -exec $(BINARY) . -v
