FROM ghcr.io/cilium/ci-kernels:6.6 AS kernel-6.6
FROM ghcr.io/cilium/ci-kernels:6.1 AS kernel-6.1
FROM ghcr.io/cilium/ci-kernels:5.15 AS kernel-5.15
FROM docker.io/library/ubuntu:22.04

ENV GOTOOLCHAIN=local
ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/lib/go-1.21/bin:$PATH

RUN mkdir -p /go

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
        && apt-get install --yes --no-install-recommends \
                qemu-system-x86 \
                qemu-system-arm \
                curl \
                ca-certificates \
                git \
                golang-1.21-go \
                gcc \
                libc6-dev \
                pkg-config \
                && apt-get clean \
                && rm -rf /var/lib/apt/lists

COPY --from=kernel-6.6 /boot/vmlinuz /kernels/vmlinuz-6.6-amd64
COPY --from=kernel-6.1 /boot/vmlinuz /kernels/vmlinuz-6.1-amd64
COPY --from=kernel-5.15 /boot/vmlinuz /kernels/vmlinuz-5.15-amd64

WORKDIR /virtrun
