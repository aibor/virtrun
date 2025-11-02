# SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
#
# SPDX-License-Identifier: GPL-3.0-or-later

# syntax=docker/dockerfile:1

FROM alpine:3.22 AS kernel

RUN apk add --no-cache tar

COPY <<"EOF" /fetch.sh
	set -ex
	mkdir -p "/kernel/$1"
	cd "/kernel/$1"
	echo "$1" > /etc/apk/arch
	apk fetch --no-cache --allow-untrusted linux-virt
	tar xf linux-virt-*.apk --wildcards --transform='s,.*/,,' \
		'boot/vmlinuz-virt' \
		'lib/modules/*/kernel/drivers/net/veth.ko.gz' \
		'lib/modules/*/kernel/drivers/net/tun.ko.gz'
	rm linux-virt-*.apk
EOF

ARG ARCH=x86_64
RUN sh /fetch.sh "$ARCH"

FROM golang:1.25-alpine3.22 AS tools

WORKDIR /tools

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go/cache \
	--mount=type=bind,source=.,target=/tools \
	go telemetry off \
	&& go install \
		github.com/jstemmer/go-junit-report/v2 \
		github.com/boumenot/gocover-cobertura

FROM alpine:3.22

LABEL org.opencontainers.image.source="https://github.com/aibor/virtrun"
LABEL org.opencontainers.image.description="Virtrun test container image"
LABEL org.opencontainers.image.licenses="GPL-3.0-or-later"

ENV VIRTRUN_ARGS="-kernel /kernel/vmlinuz-virt -transport pci"

ARG ARCH=x86_64
RUN apk add --no-cache \
	tar \
	gcc \
	musl-dev \
	"qemu-system-$ARCH"

COPY --from=kernel "/kernel/$ARCH/" /kernel/
COPY --from=tools /go/bin/ /usr/local/bin/

WORKDIR /app
