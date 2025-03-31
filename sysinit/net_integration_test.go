// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration_sysinit

package sysinit_test

import (
	"net"
	"testing"

	"github.com/aibor/virtrun/sysinit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetInterfaceUp(t *testing.T) {
	err := sysinit.SetInterfaceUp("lo")
	require.NoError(t, err)

	iface, err := net.InterfaceByName("lo")
	require.NoError(t, err, "must get interface")

	assert.NotZero(t, iface.Flags&net.FlagUp)
}
