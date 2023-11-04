#!/bin/bash
#
# Fetch pre-built kernel. See https://github.com/cilium/ci-kernels for
# available versions and architectures.
set -eEuo pipefail

: ${KERNEL_DIR:=kernel}
: ${KERNEL_VER:=${1:-6.1}}
: ${KERNEL_ARCH:=${GOARCH:-${2:-amd64}}}

kernel_file_name="$KERNEL_DIR/vmlinuz-${KERNEL_VER}-${KERNEL_ARCH}"

if [[ -e "$kernel_file_name" ]]; then
	exit 0
fi

mkdir -p "$(dirname "$kernel_file_name")"
tar \
	--file <(
		curl \
			--no-progress-meter \
			--location \
			--fail \
			"https://github.com/cilium/ci-kernels/raw/master/linux-${KERNEL_VER}-${KERNEL_ARCH}.tgz"
	) \
	--extract \
	--ignore-failed-read \
	--ignore-command-error \
	--warning=none \
	--transform="s@.*@$kernel_file_name@" \
	./boot/vmlinuz
