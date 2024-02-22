// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package qemu_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aibor/virtrun/internal/qemu"
)

func TestArgsAdd(t *testing.T) {
	a := qemu.Arguments{}
	b := qemu.UniqueArg("t").WithValue()("99")
	a.Add(b)
	assert.Equal(t, qemu.Arguments{b}, a)
}

func TestArgsBuild(t *testing.T) {
	t.Run("builds", func(t *testing.T) {
		a := qemu.Arguments{
			qemu.ArgKernel("vmlinuz"),
			qemu.ArgInitrd("boot"),
			qemu.UniqueArg("yes"),
		}
		e := []string{
			"-kernel", "vmlinuz",
			"-initrd", "boot",
			"-yes",
		}
		b, err := a.Build()
		require.NoError(t, err)
		assert.Equal(t, e, b)
	})
	t.Run("collision", func(t *testing.T) {
		a := qemu.Arguments{
			qemu.ArgKernel("vmlinuz"),
			qemu.ArgKernel("bsd"),
		}
		_, err := a.Build()
		assert.Error(t, err)
	})
}
