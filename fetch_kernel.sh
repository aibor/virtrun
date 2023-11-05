#!/bin/bash
#
# Fetch pre-built kernel. See https://github.com/cilium/ci-kernels for
# available versions and architectures.
set -eEuo pipefail

: ${KERNEL_DIR:=kernel}

version=${1:-${KERNEL_VER:-6.1}}
arch=${2:-${KERNEL_ARCH:-${GOARCH:-amd64}}}
file_name="$KERNEL_DIR/vmlinuz-${version}-${arch}"

if [[ -e "$file_name" ]]; then
	exit 0
fi

mkdir -p "$(dirname "$file_name")"
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
	--transform="s@.*@$file_name@" \
	./boot/vmlinuz
