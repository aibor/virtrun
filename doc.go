// Package pidonetest provides a simple way for running test binaries in an
// isolated system. It requires QEMU to be present on the system.
//
// It consists of two parts. One part is the library that is intended to be used
// in your go test package. The other part is the binary that is intended to be
// used with `go test -exec`.
//
// The library part is a wrapper for [testing.M.Run]. Before running the tests
// some special system file systems are mounted, like /dev, /sys, /proc, /tmp,
// /run, /sys/fs/bpf/, /sys/kernel/tracing/.
//
// In verbose mode, the kernel's tracing log will be output, which will show
// messages written with `bpf_printk`, for example.
//
// In a test package, define your custom `TestMain` function and call this
// package's Run function:
//
//	package some_test
//
//	import (
//	    "os"
//	    "testing"
//
//	    "github.com/aibor/go-pidonetest"
//	)
//
//	func TestMain(m *testing.M) {
//	    rc, err := pidonetest.Run(m)
//	    if err != nil {
//	        fmt.Printf("Error: %v", err)
//	    }
//	    os.Exit(rc)
//	}
//
// Then run the test and specify the pidonetest binary in one of the following
// ways. For now only statically compiled test work, so no cgo.
//
// If you have it installed with go install in your PATH:
//
//	$ CGO_ENABLED=0 go test -v -exec pidonetest .
//
// Or build and run on the fly with "go run":
//
//	$ CGO_ENABLED=0 go test -v -exec 'go run github.com/aibor/go-pidonetest/cmd/pidonetest' .
//
// Other architectures work as well. You need a kernel for the target
// architecture and adjust some flags for the platform. Disable KVM if your
// host architecture differs:
//
//	$ GOARCH=arm64 CGO_ENABLED=0 go test -v \
//	  -exec "pidonetest \
//	    -kernel $(realpath kernel/vmlinuz.arm64) \
//	    -qemu-bin qemu-system-aarch64 \
//	    -machine virt \
//	    -memory 128 \
//	    -cpu neoverse-n1" \
//	    -nokvm \
//	  .
package pidonetest
