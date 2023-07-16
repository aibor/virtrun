MAKEFLAGS := --no-builtin-rules
SHELL := bash
.ONESHELL:

GOBIN := $(shell realpath ./gobin)
PIDONETEST := $(GOBIN)/pidonetest

export GOBIN

.PHONY: test
test: $(PIDONETEST)
	go test -tags pidonetest -v -exec "$(PIDONETEST)" .

$(PIDONETEST):
	go install github.com/aibor/go-pidonetest/cmd/pidonetest@latest

.PHONY: testlocal
testlocal:
	go test -tags pidonetest -v -exec "go run ./cmd/pidonetest" .

.PHONY: clean
clean:
	rm -rfv "$GOBIN"
