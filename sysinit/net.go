// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

// SetInterfaceUp brings the interface with the given name up.
func SetInterfaceUp(name string) error {
	iface, err := newIfaceRequestHandle(name)
	if err != nil {
		return err
	}
	defer iface.Close()

	return iface.updateFlags(ifaceRequestFlagsSetUp)
}

// WithInterfaceUp returns a [Func] that wraps [SetInterfaceUp] and can be
// used with [Run].
func WithInterfaceUp(name string) Func {
	return func(_ *State) error {
		return SetInterfaceUp(name)
	}
}
