<!--
SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>

SPDX-License-Identifier: GPL-3.0-or-later
-->

# virtrun

[![PkgGoDev][pkg-go-dev-badge]][pkg-go-dev]
[![Go Report Card][go-report-card-badge]][go-report-card]
[![Actions][actions-test-badge]][actions-test]

virtrun allows to run a binary in a isolated QEMU guest system.

Supported architectures:
* amd64 (x86_64)
* arm64 (aarch64)
* riscv64

## Requirements

### QEMU

QEMU must be present for the architecture matching the binary. By default,
`qemu-system-x86_64`, `qemu-system-aarch64` or `qemu-system-riscv64` are used.
The architecture of the binary determines which one is used. The flag
`-qemuBin` can be used to override the default choice.

### Linux Kernel

The kernel must be compiled with support for running as guest system.
Especially, support for some kind of serial console or virtual console must be
present. All of this must be compiled into the kernel directly. Additional
kernel modules can be loaded for functionality required by the binary itself,
though, with the flag `-addModule`.

The absolute path to the kernel must be given by flag `-kernel`. Make sure the
kernel matches the architecture of your binaries and the QEMU binary.

Virtrun supports different QEMU IO transport types. Which one is needed depends
on the kernel and the QEMU machine type used. By default, the most likely
correct IO transport is chosen automatically. It can be set manually with the
flag `-transport`. For amd64 `pci` is usually the right one. For arm64 and
riscv64 it is `mmio`. `isa` can be tried as a fallback, in case there is no
output ("Error: run: guest did not print init exit code").

The Ubuntu generic kernels work out of the box and have all necessary features
compiled in.

## Usage

### Direct use

By default, virtrun brings a simple init program, that sets up the guest system
and executes the given binary. So, the binary will be a direct child of PID 1.

Usage: `virtrun [flags...] binary [args...]`

All arguments after the binary will be passed to the guest's `/init` program.
The default init program just passes them to the binary.

The following examples assume you have virtrun installed in a directory that is
in `$PATH`.

Let's use `env` as our main binary to show simple invocation and default
environment variables.

```console
$ virtrun -kernel /boot/vmlinuz-linux /usr/bin/env
HOME=/
TERM=linux
PATH=/data
```

The loopback interface is initialized by init:

```console
$ virtrun -kernel /boot/vmlinuz-linux /usr/bin/ip address
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host proto kernel_lo
       valid_lft forever preferred_lft forever
```

Additional files can be added to the guest system with the flag `-addFile`. It
can be given multiple times. Those files are added to the directory `/data`.
`PATH` is set to this directory, so binaries can be invoked easily. Also,
required the shared libraries are collected and added to the default library
directory as well:

The `tree` binary can be used to inspect the guest's file system.

```console
$ virtrun -kernel /boot/vmlinuz-linux -addFile /usr/bin/bash /usr/bin/tree -x
.
|-- data
|   `-- bash
|-- dev
|-- init
|-- lib
|   |-- ld-linux-x86-64.so.2
|   |-- libc.so.6
|   |-- libncursesw.so.6
|   |-- libreadline.so.8
|   `-- modules
|-- lib64 -> /lib
|-- main
|-- proc
|-- root
|-- run
|-- sys
|-- tmp
`-- usr
    `-- lib -> /lib
```

Kernel modules can be added with the flag `-addModule` that can be used
multiple times. The modules are added to the directory `/lib/modules` and are
loaded automatically by the default init in the order they are given in the
command line. Dependencies must be provided and are not resolved automatically.
The modules must be added in the correct order.

### With `go test -exec`

Virtrun can be used to run go tests in a clean and isolated environment.
Additionally, it allows to test for a different architecture or kernel. It can
be used with the go test's `-exec` flag. Just pass the complete virtrun
invocation as string to that flag. It will be invoked for each test binary.

Since go test changes into the package directory for running the test, absolute
paths must be used for any file path that is passed to virtrun by flag
(`-kernel`, `-addFile`, `qemuBin`, ...).

Installed into `$PATH`:

```console
$ go test -exec "virtrun -kernel /boot/vmlinuz-linux" .
```

Not installed into `$PATH`:

```console
$ go test -exec "go run /go/bin/virtrun -kernel /boot/vmlinuz-linux" .
```

All flags can also be passed by environment variable `VIRTRUN_ARGS`:

```console
$ export VIRTRUN_ARGS="-kernel /boot/vmlinuz-linux"
$ go test -exec virtrun .
```

Run cross compiled test:

```console
$ export VIRTRUN_ARGS="-kernel /absolute/path/to/vmlinuz-arm64"
$ GOARCH=arm64 go test -exec virtrun .
```

Virtrun supports some go test flags that set output files, like coverage or
resource profile files, and uses virtual consoles to write the content from the
guest system back to the host:

```console
$ go test -exec virtrun -cover -coverprofile cover.out .
```

For debugging, use virtrun's flags `-verbose` and `-debug` together with go
test's flag `-v`:

```console
$ go test -exec "virtrun -verbose -debug" -v .
```

### Standalone mode

In Standalone mode, the given binary is executed as `/init` directly. For this
to work, your binary must do any system setup itself. However, the only
essential required task is to communicate the exit code on stdout and shutdown
the system.

The sub-package [sysinit](https://pkg.go.dev/github.com/aibor/virtrun/sysinit)
provides helper functions for the necessary tasks.

A simple init can be built using `sysinit.Run` which is the main entry point 
for an init system. It runs user provided functions and shuts down the system 
on termination. For an example, see the
[simple init program](internal/virtrun/init/cmd/main.go) that is embedded in 
the virtrun binary and is the init used in the default wrapped mode.

## Internals

### Work flow

For running the QEMU command an initramfs archive file must be built. For this,
the main binary is copied to `/main` and all additional files are copied into
the `/data/` directory. For those files all required dynamic libraries are
added into the `/lib/` directory. Kernel modules are copied into the
`/lib/modules/` directory.

The build archive file is used for running the QEMU command along with the
given kernel file. Before the run is executed, go test flags that provide file
paths are rewritten, so the guest writes into serial consoles and the host
forwards them into the actual files given by the user.

### Exit Code Communication

Virtrun wraps QEMU and runs an init program that runs and communicates its exit
code via a defined formatted string on stdout that is parsed by the virtrun.
Everything else on stdout is printed directly as is.

### File Output

For writing into files on the host (like for go test profiles), a dedicated
virtual console is set up for each file.

### Architecture Detection

The given main binary determines the architecture that is used for setting 
the defaults. The QEMU executable, machine type and transport type are set
based on the main binaries architecture if not given explicitly by flags. KVM
is enabled if present and accessible and not disabled explicitly. See 
`virtrun -help` for all flags.

[pkg-go-dev]:           https://pkg.go.dev/github.com/aibor/virtrun
[pkg-go-dev-badge]:     https://pkg.go.dev/badge/github.com/aibor/virtrun
[go-report-card]:       https://goreportcard.com/report/github.com/aibor/virtrun
[go-report-card-badge]: https://goreportcard.com/badge/github.com/aibor/virtrun
[actions-test]:         https://github.com/aibor/virtrun/actions/workflows/test.yaml
[actions-test-badge]:   https://github.com/aibor/virtrun/actions/workflows/test.yaml/badge.svg?branch=main
