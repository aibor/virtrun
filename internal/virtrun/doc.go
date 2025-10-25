// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Package virtrun provides the main utilities to build run a binary in a
// short lived QEMU guest system. The kernel and required modules as well as
// the QEMU binary, and the main binary must be provided by the user. The
// main binary and modules are copied into a transient initramfs.
//
// A simple init that does basic system setup (mount usual virtual file systems,
// load modules, bring up loopback interface), runs the main binary, and
// communicates its exit code to the host is provided by virtrun. It is
// pre-compiled and based on the package [github.com/aibor/virtrun/sysinit].
// Users may use their own init by passing a standalone main binary that
// implements the required features which are just graceful guest termination
// and communicating the exit code beforehand.
package virtrun
