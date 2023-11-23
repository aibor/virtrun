FROM docker.io/library/ubuntu:22.04

ENV GOTOOLCHAIN=local
ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/lib/go-1.18/bin:$PATH

RUN mkdir -p /go /virtrun

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
	&& apt-get install --yes --no-install-recommends \
		qemu-system-x86 \
		qemu-system-arm \
		curl \
		ca-certificates \
		git \
		golang-1.18-go \
		gcc \
		libc6-dev \
		pkg-config \
		&& apt-get clean \
		&& rm -rf /var/lib/apt/lists

COPY fetch_kernel.sh /fetch_kernel.sh

WORKDIR /virtrun
