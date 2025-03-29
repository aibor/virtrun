# SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
#
# SPDX-License-Identifier: GPL-3.0-or-later

# syntax=docker/dockerfile:1

FROM alpine:3.21 AS kernel

RUN apk add --no-cache tar

COPY <<"EOF" /fetch.sh
	set -ex
	mkdir -p "/kernel/$1"
	cd "/kernel/$1"
	echo "$2" > /etc/apk/arch
	apk fetch --no-cache --allow-untrusted linux-virt
	tar xf linux-virt-*.apk --wildcards --transform='s,.*/,,' \
		'boot/vmlinuz-virt' \
		'lib/modules/*/kernel/drivers/net/veth.ko.gz' \
		'lib/modules/*/kernel/drivers/net/tun.ko.gz'
	rm linux-virt-*.apk
EOF

RUN sh /fetch.sh amd64 x86_64 && sh /fetch.sh arm64 aarch64

FROM golang:1.24-alpine

LABEL org.opencontainers.image.source="https://github.com/aibor/virtrun"
LABEL org.opencontainers.image.description="Virtrun test container image"
LABEL org.opencontainers.image.licenses="GPL-3.0-or-later"

ENV VIRTRUN_ARGS="-kernel /kernel/amd64/vmlinuz-virt -transport pci"

RUN apk add --no-cache \
	tar \
	gcc \
	musl-dev \
	qemu-system-x86_64 \
	qemu-system-aarch64

COPY --from=kernel /kernel/amd64/ /kernel/amd64/
COPY --from=kernel /kernel/arm64/ /kernel/arm64/

WORKDIR /tools

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
	--mount=type=bind,source=.github/workflows,target=/tools \
	go install github.com/jstemmer/go-junit-report/v2 \
	&& go install github.com/boumenot/gocover-cobertura

WORKDIR /app

RUN --mount=type=bind,source=.,target=/app \
	go mod download -x
