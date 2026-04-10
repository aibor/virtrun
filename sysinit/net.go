// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"fmt"
	"net/netip"
)

// ConfigureInterface configures the interface with the given name by bringing
// it up and configuring it with the given IPv4 address and netmask if set.
//
// The interface name must match the kernel's requirements on a valid name. That
// is, it must not be longer than 15 characters.
func ConfigureInterface(name string, prefix netip.Prefix) error {
	iface, err := newIfaceRequestHandle(name)
	if err != nil {
		return err
	}
	defer iface.Close()

	err = iface.updateFlags(ifaceRequestFlagsSetUp)
	if err != nil {
		return fmt.Errorf("set iface flags: %w", err)
	}

	// Exit early if no address is set.
	if !prefix.IsValid() {
		return nil
	} else if !prefix.Addr().Is4() {
		return fmt.Errorf("%w: address not IPv4", ErrInvalidConfig)
	}

	err = iface.setAddr(ifreqAddrLocal, prefix.Addr().AsSlice())
	if err != nil {
		return fmt.Errorf("set addr: %w", err)
	}

	ipv4AllSys := netip.AddrFrom4([4]byte{255, 255, 255, 255})
	network := netip.PrefixFrom(ipv4AllSys, prefix.Bits()).Masked()

	err = iface.setAddr(ifreqAddrNetmask, network.Addr().AsSlice())
	if err != nil {
		return fmt.Errorf("set netmask: %w", err)
	}

	return nil
}

// WithConfiguredInterfaces returns a [Func] that runs [ConfigureInterface] for
// all interfaces given.
func WithConfiguredInterfaces(ifaces *map[string]netip.Prefix) Func {
	return func(_ *State) error {
		for name, cfg := range *ifaces {
			err := ConfigureInterface(name, cfg)
			if err != nil {
				return fmt.Errorf("configure %s: %w", name, err)
			}
		}

		return nil
	}
}
