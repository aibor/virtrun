// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import "fmt"

// Poweroff shuts down the system.
//
// It does not return, unless in case of error. It should be called deferred at
// the start of the main init function.
func Poweroff() error {
	// Use restart instead of poweroff for shutting down the system since it
	// does not require ACPI. The guest system should be started with noreboot.
	if err := reboot(); err != nil {
		return fmt.Errorf("poweroff failed: %w", err)
	}

	return nil
}

// IsPidOne returns true if the running process has PID 1.
func IsPidOne() bool {
	return getpid() == 1
}

// Sysctl allows to set kernel knobs via sysctl(2).
func Sysctl(key, value string) error {
	return sysctl(key, value)
}
