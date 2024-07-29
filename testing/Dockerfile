# SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
#
# SPDX-License-Identifier: MIT
#
# syntax=docker/dockerfile:1

FROM alpine:3.20

ENV ARCH=amd64
RUN --mount=type=cache,target=/var/cache/apk \
  apk add \
    go \
    qemu-system-x86_64 \
    qemu-system-aarch64

VOLUME /data
WORKDIR /data
