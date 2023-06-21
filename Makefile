MAKEFLAGS := --no-builtin-rules
SHELL := bash
.ONESHELL:

GOBIN := $(shell realpath ./gobin)
PIDONETEST := $(GOBIN)/pidonetest

export GOBIN

.PHONY: test
test: $(PIDONETEST)
	go test -v -exec "$(PIDONETEST) -debug" .

$(PIDONETEST):
	go install github.com/aibor/go-pidonetest/cmd/pidonetest@latest

.PHONY: testlocal
testlocal:
	go test -v -exec "go run ./cmd/pidonetest -debug" .

.PHONY: clean
clean:
	rm -rfv "$GOBIN"
