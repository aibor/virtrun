#!/bin/bash

set -eEuo pipefail

kernel_version=${1:?missing kernel version}
kernel_arch=${2:?missing kernel arch}
mode=${3:?mode missing}

export KERNEL_DIR=kernel

rundir="$(dirname "${BASH_SOURCE[0]}")"
"$rundir"/fetch_kernel.sh $kernel_version $kernel_arch

go install -buildvcs=false ./cmd/virtrun

kernel_path="$KERNEL_DIR/vmlinuz-${kernel_version}-${kernel_arch}"
virtrun_args=("-kernel" "$(realpath $kernel_path)")
test_tags=selftest

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

GOARCH=$kernel_arch go test \
	-v \
	-timeout 2m \
	-exec "virtrun ${virtrun_args[*]} -verbose" \
	-tags "$test_tags" \
	-cover \
	-coverprofile /tmp/cover.out \
	-coverpkg github.com/aibor/virtrun \
	./selftest
