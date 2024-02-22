// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package sysinit

import (
	"fmt"

	"github.com/vishvananda/netlink"
)

// ConfigureLoopbackInterface brings the loopback interface up. Kernel should configure
// address already automatically.
func ConfigureLoopbackInterface() error {
	link, err := netlink.LinkByName("lo")
	if err != nil {
		return fmt.Errorf("get link: %v", err)
	}

	return netlink.LinkSetUp(link)
}
