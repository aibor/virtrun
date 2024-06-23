<!--
SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>

SPDX-License-Identifier: MIT
-->

# virtrun

[![PkgGoDev][pkg-go-dev-badge]][pkg-go-dev]
[![Go Report Card][go-report-card-badge]][go-report-card]

virtrun is a library and binary QEMU wrapper for running binaries in an
isolated system.

The package uses itself for testing, so see the guest tests in
[integrationtesting](integrationtesting/) for real life examples.


## Requirements

### QEMU

The binaries for running QEMU for the architecture matching the kernel and
binary you want to use, either `qemu-system-x86_64` or `qemu-system-aarch64`.
If not in `$PATH`, you can specify the path to the binary with the flag
`-qemu-bin`.

### Linux Kernel

The kernel must be compiled to work with some support for working in QEMU.
Especially some kind of serial console or virtual console must be present. All
of this must be compiled into the kernel directly, ass there is no way to load
kernel modules, unless your given binary does that.

The absolute path to the kernel must be given by flag `-kernel`. Make sure the
kernel matches the architecture of your binaries and the QEMU binary.

By default, the most likely correct IO transport is chosen automatically. It
can be set manually with the flag `-transport`. With x86 `pci` is usually the
right now. With arm64 it is `mmio`. `isa` can be tried as a fallback, in case
there is no output ("Error: run: guest did not print init return code").

The Ubuntu kernels work out of the box and have all necessary features compiled
in.

## Usage

### Direct use

By default, virtrun brings a simple init program, that sets up the guest system
and then executes your binary. So, your binary will be a direct child of PID 1.

Usage: `virtrun [flags...] binary [args...]`

All arguments after the binary will be passed to the guest's
`/init` program. The default init program will pass them to the binary.

The following examples assume you have virtrun installed in a directory that is
in `$PATH`. Instead, you can can also use `go run github.com/aibor/virtrun`.

Let's use `env` as our main binary to show simple invocation and default
environment variables.

```console
$ virtrun -kernel /boot/vmlinuz-linux /usr/bin/env
HOME=/
TERM=linux
PATH=/data
```

Loopback interface is initialized by init:

```console
$ virtrun -kernel /boot/vmlinuz-linux /usr/bin/ip address
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host proto kernel_lo
       valid_lft forever preferred_lft forever
```

Additional files can be added to the guest system with the flag `-addFile` that
can be used multiple times. The files are added into the directory `/data`.
`PATH` is set to this directory, so binaries can be invoked easily. Also,
required shared libraries are added to the ELF interpreter's default library
directory as well:

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
|   `-- libreadline.so.8
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

### With `go test -exec`

Virtrun can be used to run go tests that require root privileges, have special
system environment requirements, test for a different architecture or kernel
version. It can be used to wrap go test binary execution by using it with the
go test's `-exec` flag. Just pass the complete virtrun invocation as string to
that flag. It will be invoked for each test binary.

Since go test changes into the package directory for running the test, absolute
paths must be used for any file path that is passed to virtrun by flag
(`-kernel`, `-addFile`, `qemu-bin`, ...).

Use without installation:

```console
$ go test -exec "go run github.com/aibor/virtrun -kernel /boot/vmlinuz-linux" .
```

Installed into `$PATH`:

```console
$ go test -exec "virtrun -kernel /boot/vmlinuz-linux" .
```

Setting flags by environment variable:

```console
$ export VIRTRUN_ARGS="-kernel /boot/vmlinuz-linux"
$ go test -exec virtrun .
```

Use arm64 kernel on a amd64 host:

```console
$ export VIRTRUN_ARGS="-kernel /absolute/path/to/vmlinuz-arm64"
$ GOARCH=arm64 go test -exec virtrun .
```

Virtrun supports go test flags that set output files, like coverage or resource
profile files, and uses virtual consoles to send the content from the guest
system back to the host:

```
$ go test -exec 'virtrun' -cover -coverprofile cover.out .
```

For debugging, use virtrun's flag `-verbose` togehter with go test's flag `-v`:

```console
$ go test -exec "virtrun -verbose" -v .
```

### Standalone mode

In Standalone mode, your given binary is executed as `/init` directly. For this
to work, your binary must do system setup itself. The only essential required
task it has is to communicate the exit code on stdout and shutdown the system.

The sub-package [sysinit](https://pkg.go.dev/github.com/aibor/virtrun/sysinit)
provides helper functions for necessary tasks.

A simple init can be built using `sysinit.Run` which is a wrapper for those
essential tasks. See the [simple init program](internal/initprog/init/main.go)
that is used in the default wrapped mode, for inspiration.

For go test binaries `sysinit.RunTests` can be used in a custom `TestMain`
function if you need to do any additional set up for your test run. It is is a
wrapper for `sysinit.Run` around `testing.M.Run`.

So, in a test package, define your custom TestMain function and call
`sysinit.RunTests`. You may keep this in a separate test file and use build
constraints in order to have an easy way of separating such test from normal go
tests that can run on the same system:

```
//go:build virtrun

package some_test

import (
    "testing"

    "github.com/aibor/virtrun/sysinit"
)

func TestMain(m *testing.M) {
    sysinit.RunTests(m)
}
```

See the integration_tests/guest directory for a working example.

Instead of using `sysinit.RunTests` you can use call the various parts
individually, of course, and just mount the file systems you need or additional
ones. See `sysinit.RunTests` for the steps it does.

## Internals

### Return Code Communication

Virtrun wraps QEMU and runs an init program that runs and communicates its exit
code via a defined formatted string on stdout that is parsed by the virtrun.
Everything else on stdout is printed directly as is.

### File Output

For writing into files on the host (like for go test profiles), a dedicated
virtual console is set up for each file.

### Architecture Detection

Depending on the presence of the environment variables `VIRTRUN_ARCH`,
`GOARCH`, or with the runtime arch, the correct qemu-system binary and machine
type is used. KVM is enabled if present and accessible. Those things can be
overridden by flags. See `virtrun -help` for all flags.

Virtrun supports different QEMU IO transport types. Which is needed depends on
the kernel and machine type used. If you don't get any output, try different
transport types with flag `-transport`

[pkg-go-dev]:           https://pkg.go.dev/github.com/aibor/virtrun
[pkg-go-dev-badge]:     https://pkg.go.dev/badge/github.com/aibor/virtrun
[go-report-card]:       https://goreportcard.com/report/github.com/aibor/virtrun
[go-report-card-badge]: https://goreportcard.com/badge/github.com/aibor/virtrun
