# virtrun

[![PkgGoDev](https://pkg.go.dev/badge/github.com/aibor/virtrun)](https://pkg.go.dev/github.com/aibor/virtrun)
virtrun is a library and binary QEMU wrapper for running binaries in an 
isolated system as PID 1.

The package uses itself for testing, so see 
[integrationtesting](integrationtesting/) for real life examples.

## Quick start

The easiest way to use virtrun is to use the embedded init programs. With this, 
no init binary needs to be provided. 

The path to the kernel must be given, either by flag `-kernel` or by the 
environment variable `QEMU_KERNEL`. Make sure the kernel matches the 
architecture of the binaries and the QEMU binary.

If you have it installed in your PATH, run a go test like this:

```
$ go test -exec "virtrun -kernel /boot/vmlinuz-linux" .
$ virtrun -kernel /boot/vmlinuz-linux /usr/bin/env
```

Or use directly:

```
$ go test -exec "go run github.com/aibor/virtrun" .
```

Other architectures work as well. You need a kernel for the target
architecture. Currently supported are "amd64" and "arm64".

## How it works

Virtrun wraps QEMU running an init that runs and communicates its exit code 
via stdout back to virtrun. It is intended to run simple binaries like 
go test binaries. Virtrun tries to provide all required linked libraries of the
binaries in the QEMU guest.

Depending on the presence of GOARCH or the runtime arch, the correct
qemu-system binary and machine type is used. KVM is enabled if present and
accessible. Those things can be overridden by flags. See "virtrun -help"
for all flags.

Virtrun supports different QEMU IO transport types. Which is needed depends on 
the kernel and machine type used. If you don't get any output, try different
transport types with flag `-transport`

All flags that are given after the binaries will be passed to the guest's 
`/init` program. There is special handling for file related gotestflags. Those
are rewritten to virtual consoles before they are passed. So, gotestflags like 
coverprofile can be used.

## Standalone mode

In Standalone mode, the first given binary is required to be able to act as a 
system init binary. The only essential required functions are to communicate 
the exit code on stdout and shutdown the system. 

A simple init can be built using `sysinit.Run` which is a wrapper for those 
essential tasks.

For go test binaries this can be done by using `sysinit.RunTests` in a custom
`TestMain` function. It is is a wrapper for `testing.M.Run`. Before running the 
tests some special system file systems are mounted and handles communicating 
the return code.

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

With the `TestMain` function in place, run the test and specify the virtrun
binary in one of the following ways:

```
$ go test -tags virtrun -exec 'virtrun -standalone' .
$ go test -tags virtrun -exec 'virtrun -standalone' -cover -coverprofile cover.out .
```
