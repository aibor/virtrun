// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

// ConfigureLoopbackInterface brings the loopback interface up.
//
// Kernel configures addresses automatically.
func ConfigureLoopbackInterface() error {
	return SetInterfaceUp("lo")
}

// SetInterfaceUp brings the interface with the given name up.
func SetInterfaceUp(name string) error {
	return setInterfaceUp(name)
}
