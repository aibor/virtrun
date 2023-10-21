MAKEFLAGS := --no-builtin-rules
SHELL := bash
.ONESHELL:

GOBIN := $(shell realpath ./gobin)
PIDONETEST := $(GOBIN)/pidonetest

export GOBIN

$(PIDONETEST):
	go install github.com/aibor/go-pidonetest/cmd/pidonetest@main

.PHONY: test-installed-wrapped
test-installed-wrapped: $(PIDONETEST)
	go test -v -exec "$(PIDONETEST) -wrap" .

.PHONY: test-installed
test-installed: $(PIDONETEST)
	go test -tags pidonetest -v -exec "$(PIDONETEST)" .

.PHONY: test-local-wrapped
test-local-wrapped:
	go test -v -exec "go run ./cmd/pidonetest -wrap" .

.PHONY: test-local
test-local:
	go test -tags pidonetest -v -exec "go run ./cmd/pidonetest" .

.PHONY: clean
clean:
	rm -rfv "$(GOBIN)"
