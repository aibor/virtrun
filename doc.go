// Package pidonetest provides a simple way for running go tests in an isolated
// system. It requires QEMU to be present on the system.
//
// Depending on the presence of GOARCH or the runtime arch, the correct
// qemu-system binary and machine type is used. KVM is enabled if present and
// accessible. Those things can be overridden by flags. See "pidonetest -help"
// for all flags.
//
// # Default mode
//
// The easiest way to use this package is to use the default mode. This does
// not need any special support by your test package. The downside is, that it
// can only be used if the target architecture matches the pidonetest binary
// architecture. See "Library" if you want to test for different architectures.
//
// If the kernel used does not have support for Virtio-MMIO compiled in, use
// flag "-transport 0" in order to use legacy isa-pci serial consoles for IO.
//
// Set absolute path to the kernel to use by environment variable:
//
//	$ export QEMU_KERNEL=/boot/vmlinuz-linux
//
// If you have it installed with go install in your PATH:
//
//	$ go test -exec "pidonetest" .
//
// Or build and run on the fly with "go run":
//
//	$ go test -exec 'go run github.com/aibor/pidonetest/cmd/pidonetest' .
//
// There is also support for coverage profiles. Just specify it as usual:
//
//	$ go test -exec "pidonetest" -cover -coverprofile cover.out .
//
// # Standalone mode
//
// In Standalone mode, the test binary can be used as init binary directly.
// For this to work, the init steps need to be compiled in. this is done by
// defining your own TestMain function using the provided package as library.
//
// The library part is a wrapper for [testing.M.Run]. Before running the tests
// some special system file systems are mounted, like /dev, /sys, /proc, /tmp,
// /run, /sys/fs/bpf/, /sys/kernel/tracing/.
//
// In a test package, define your custom TestMain function and call
// [pidonetest.RunTests]. You may keep this in a separate test file and use
// build constraints in order to have an easy way of separating such test from
// normal go tests that can run on the same system:
//
//	//go:build pidonetest
//
//	package some_test
//
//	import (
//	    "testing"
//
//	    "github.com/aibor/pidonetest"
//	)
//
//	func TestMain(m *testing.M) {
//	    pidonetest.RunTests(m)
//	}
//
// See the selftest directory for a working example.
//
// Instead of using [pidonetest.RunTests] you can use call the various parts
// individually, of course, and just mount the file systems you need or
// additional ones. See [pidonetest.RunTests] for the steps it does.
//
// With the TestMain function in place, run the test and specify the pidonetest
// binary in one of the following ways. If the test binary is dynamically linked
// libraries are resolved. However, if you run into errors try again with
// "CGO_ENABLED=0", if you don't need cgo. In any case, make sure the QEMU
// binary used has the same architecture as the test binary.
//
// If you have it installed with go install in your PATH:
//
//	$ go test -tags pidonetest -exec 'pidonetest -standalone' .
//
// Or build and run on the fly with "go run":
//
//	$ go test -tags pidonetest -exec 'go run github.com/aibor/pidonetest/cmd/pidonetest -standalone' .
//
// There is also support for coverage profiles. Just specify it as usual:
//
//	$ go test -tags pidonetest -exec 'pidonetest -standalone' -cover -coverprofile cover.out .
//
// Other architectures work as well. You need a kernel for the target
// architecture.
//
//	$ GOARCH=arm64 go test -exec "pidonetest -kernel $(realpath kernel/vmlinuz.arm64) -standalone" .
package pidonetest
