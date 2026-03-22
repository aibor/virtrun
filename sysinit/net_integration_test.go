// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration_sysinit

package sysinit_test

import (
	"net"
	"net/netip"
	"testing"

	"github.com/aibor/virtrun/sysinit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureInterface(t *testing.T) {
	prefix := netip.MustParsePrefix("10.0.0.1/24")
	err := sysinit.ConfigureInterface("lo", prefix)
	require.NoError(t, err)

	iface, err := net.InterfaceByName("lo")
	require.NoError(t, err, "must get interface")

	assert.NotZero(t, iface.Flags&net.FlagUp, "flag up")

	actualAddrs, err := iface.Addrs()
	require.NoError(t, err, "must get addrs")

	expectedAddrs := []net.Addr{
		&net.IPNet{
			IP:   net.IP(prefix.Addr().AsSlice()).To16(),
			Mask: net.CIDRMask(24, 32),
		},
		&net.IPNet{
			IP:   net.IPv6loopback,
			Mask: net.CIDRMask(128, 128),
		},
	}

	t.Log(actualAddrs)

	assert.Equal(t, expectedAddrs, actualAddrs)
}
