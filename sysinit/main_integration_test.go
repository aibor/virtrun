// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package sysinit_test

import (
	"testing"

	"github.com/aibor/virtrun/sysinit"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	cfg := sysinit.Config{
		MountPoints: sysinit.MountPoints{
			"/proc": {
				FSType: sysinit.FSTypeProc,
			},
		},
	}

	sysinit.Main(cfg, func() (int, error) {
		return m.Run(), nil
	})
}

func TestIsPidOne(t *testing.T) {
	assert.True(t, sysinit.IsPidOne())
}

func TestIsPidOneChild(t *testing.T) {
	assert.False(t, sysinit.IsPidOneChild())
}
