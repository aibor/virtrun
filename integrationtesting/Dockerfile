FROM docker.io/library/golang:1.22.0-bookworm

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
        && apt-get install --yes --no-install-recommends \
                qemu-system-x86 \
                qemu-system-arm \
                && apt-get clean \
                && rm -rf /var/lib/apt/lists
