// Package pidonetest provides a simple way for running go tests in an isolated
// system. It requires QEMU to be present on the system.
//
// # Quick Start
//
// The easiest way to use this package is to use the "wrapper mode". This does
// not need any special support by your test package. The downside is, that it
// can only be used if the target architecture matches the pidonetest binary
// architecture. See "Library" if you want to test for different architectures.
//
// If you have it installed with go install in your PATH:
//
//	$ go test -exec "pidonetest -wrap" .
//
// Or build and run on the fly with "go run":
//
//	$ go test -exec 'go run github.com/aibor/go-pidonetest/cmd/pidonetest -wrap' .
//
// There is also support for coverage profiles. Just specify it as usual:
//
//	$ go test -exec "pidonetest -wrap" -cover -coverprofile cover.out .
//
// # Library
//
// It consists of two parts. One part is the library that is intended to be used
// in your go test package. The other part is the binary that is intended to be
// used with "go test -exec".
//
// The library part is a wrapper for [testing.M.Run]. Before running the tests
// some special system file systems are mounted, like /dev, /sys, /proc, /tmp,
// /run, /sys/fs/bpf/, /sys/kernel/tracing/.
//
// In a test package, define your custom TestMain function and call this
// package's [Run] function. You may keep this in a separate test file and use
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
//	    "github.com/aibor/go-pidonetest"
//	)
//
//	func TestMain(m *testing.M) {
//	    pidonetest.Run(m)
//	}
//
// Instead of using [Run] you can use call the various parts individually, of
// course, and just mount the file systems you need or additional ones. See
// [Run] for the steps it does.
//
// With the TestMain function in place, run the test and specify the pidonetest
// binary in one of the following ways. If the test binary is dynamically linked
// libraries are resolved. However, if you run into errors try again with
// "CGO_ENABLED=0", if you don't need cgo. In any case, make sure the QEMU
// binary used has the same architecture as the test binary.
//
// If you have it installed with go install in your PATH:
//
//	$ go test -tags pidonetest -exec pidonetest .
//
// Or build and run on the fly with "go run":
//
//	$ go test -tags pidonetest -exec 'go run github.com/aibor/go-pidonetest/cmd/pidonetest' .
//
// There is also support for coverage profiles. Just specify it as usual:
//
//	$ go test -tags pidonetest -exec pidonetest -cover -coverprofile cover.out .
//
// Other architectures work as well. You need a kernel for the target
// architecture and adjust some flags for the platform. Disable KVM if your
// host architecture differs:
//
//	$ GOARCH=arm64 go test -v \
//	  -exec "pidonetest \
//	    -kernel $(realpath kernel/vmlinuz.arm64) \
//	    -qemu-bin qemu-system-aarch64 \
//	    -machine virt \
//	    -memory 128 \
//	    -cpu neoverse-n1" \
//	    -nokvm \
//	  .
//
// See "pidonetest -help" for all flags.
package pidonetest
