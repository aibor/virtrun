# SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
#
# SPDX-License-Identifier: MIT
#
# syntax=docker/dockerfile:1

FROM alpine:3.19

 # link dynamic linker so the test with prebuilt testdata works
RUN set -eux; \
  mkdir /lib64; \
  ln -s /lib/ld-musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2;

ENV ARCH=amd64
RUN --mount=type=cache,target=/var/cache/apk \
  apk add \
    "$pkg" \
	go \
    qemu-system-x86_64 \
    qemu-system-aarch64

VOLUME /data
WORKDIR /data
