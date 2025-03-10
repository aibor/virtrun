// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package sysinit_test

import (
	"net"
	"testing"

	"github.com/aibor/virtrun/sysinit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureLoopbackInterface(t *testing.T) {
	err := sysinit.ConfigureLoopbackInterface()
	require.NoError(t, err)

	iface, err := net.InterfaceByName("lo")
	require.NoError(t, err, "must get interface")

	assert.Positive(t, iface.Flags&net.FlagUp)

	addrs, err := iface.Addrs()
	require.NoError(t, err, "must get addresses")

	require.Len(t, addrs, 2, "should have 2 addresses")

	assert.Equal(t, "127.0.0.1/8", addrs[0].String())
	assert.Equal(t, "::1/128", addrs[1].String())
}
