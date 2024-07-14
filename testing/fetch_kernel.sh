#!/bin/sh

# SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
#
# SPDX-License-Identifier: MIT

#
# Fetch kernel from https://github.com/cilium/ci-kernels container registry.
# They build tests kernels for amd64 and arm64 since linux 6.7. The images
# are tagged with major version like "6.7", but also with minor versions like
# "6.7.1". The kernel file is always "/boot/vmlinuz".

: ${CONTAINERBIN:=podman}

kernel_version=${1:?kernel version missing}
kernel_arch=${2:?kernel arch missing}
dest=${3:?destination path missing}

container_id=$($CONTAINERBIN create --platform="linux/$kernel_arch" "ghcr.io/cilium/ci-kernels:$kernel_version" sh)
trap "$CONTAINERBIN rm $container_id" EXIT

$CONTAINERBIN cp "$container_id:/boot/vmlinuz" "$dest"
