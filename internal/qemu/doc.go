// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Package qemu provides utilities for composing and running QEMU system
// virtualization commands as needed by virtrun. It expects the required QEMU
// binary to be present on the system.
//
// The guest system is expected to send kernel output and error messages on the
// default console (e.g. /dev/hvc0). Stdout and additional optional output is
// supposed to be sent on a separate [pipe.Pipe] (e.g. /dev/virtrun1 for stdout,
// /dev/virtrun2 for optional file.)
//
// The quest system is expected to communicate the exit code of it's main binary
// via a magic string on the default output.
package qemu
