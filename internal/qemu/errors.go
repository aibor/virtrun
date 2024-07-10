// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package qemu

import "errors"

var (
	// ErrGuestNoRCFound is returned if no return code matching the [RCFmt] is
	// found and no other error is found.
	ErrGuestNoRCFound = errors.New("guest did not print init return code")
	// ErrGuestPanic is returned if a kernel panic occurred in the guest
	// system.
	ErrGuestPanic = errors.New("guest system panicked")
	// ErrGuestOom is returned if the guest system ran out of memory.
	ErrGuestOom = errors.New("guest system ran out of memory")
)
