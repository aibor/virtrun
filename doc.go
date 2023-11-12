// Package virtrun provides a simple way for running binaries in an isolated
// system. It requires QEMU to be present on the system.
//
// The sysinit sub package provides functions for building a simple init binary
// that sets up system virtual file system mount points, sets up correct
// shutdown and communicates the binaries exit codes on stdout for consumption
// by the QEMU wrapper virtrun.
package virtrun
