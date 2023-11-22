#!/bin/bash

mode=${1:?mode missing}

GOARCH= go install -buildvcs=false ./cmd/virtrun

# KERNEL provided by container.
virtrun_args=("-kernel" "$KERNEL")
test_tags=integration

case "$mode" in
wrapped) ;;
standalone)
	virtrun_args+=("-standalone")
	test_tags+=,standalone
	;;
*)
	echo "unknown mode"
	exit 1
	;;
esac

go test \
	-v \
	-timeout 2m \
	-exec "virtrun ${virtrun_args[*]} -verbose" \
	-tags "$test_tags" \
	-cover \
	-coverprofile /tmp/cover.out \
	-coverpkg github.com/aibor/virtrun/sysinit \
	./integration_tests/guest/...
