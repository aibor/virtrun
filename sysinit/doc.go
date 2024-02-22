// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

// The sysinit sub package provides functions for building a simple init binary
// that sets up system virtual file system mount points, sets up correct
// shutdown and communicates the binaries exit codes on stdout for consumption
// by the QEMU wrapper virtrun.
package sysinit
