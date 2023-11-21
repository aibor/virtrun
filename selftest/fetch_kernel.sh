#!/bin/bash
#
# Fetch pre-built kernel. See https://github.com/cilium/ci-kernels for
# available versions and architectures.
set -eEuo pipefail

: ${KERNEL_DIR:=kernel}

version=${1:-${KERNEL_VER:-6.1}}
arch=${2:-${KERNEL_ARCH:-${GOARCH:-amd64}}}
file_name="vmlinuz-${version}-${arch}"

if [[ ! -e "$KERNEL_DIR/$file_name" ]]; then
	mkdir -p "$KERNEL_DIR"

	tar \
		--file <(
			curl \
				--no-progress-meter \
				--location \
				--fail \
				"https://github.com/cilium/ci-kernels/raw/master/linux-${version}-${arch}.tgz"
		) \
		--extract \
		--ignore-failed-read \
		--ignore-command-error \
		--warning=none \
		--transform="s@.*@$KERNEL_DIR/$file_name@" \
		./boot/vmlinuz
fi

echo "$(realpath "$KERNEL_DIR/$file_name")"
