# SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
#
# SPDX-License-Identifier: MIT

FROM docker.io/library/golang:1.22.0-bookworm AS build-stage

WORKDIR /virtrun

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go generate ./...
RUN CGO_ENABLED=0 GOOS=linux go build -o /virtrun

FROM docker.io/library/ubuntu:22.04 AS run-stage

COPY --from=build-stage /virtrun /usr/local/bin/

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
	&& apt-get install --yes --no-install-recommends \
		qemu-system-x86 \
		qemu-system-arm \
	&& apt-get clean \
	&& rm -rf /var/lib/apt/lists
