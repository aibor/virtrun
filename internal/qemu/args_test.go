// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package qemu_test

import (
	"testing"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildArgumentStrings(t *testing.T) {
	t.Run("builds", func(t *testing.T) {
		a := []qemu.Argument{
			qemu.UniqueArg("kernel", "vmlinuz"),
			qemu.UniqueArg("initrd", "boot"),
			qemu.UniqueArg("yes"),
			qemu.RepeatableArg("more", "a"),
			qemu.RepeatableArg("more", "b"),
		}
		e := []string{
			"-kernel", "vmlinuz",
			"-initrd", "boot",
			"-yes",
			"-more", "a",
			"-more", "b",
		}
		b, err := qemu.BuildArgumentStrings(a)
		require.NoError(t, err)
		assert.Equal(t, e, b)
	})

	t.Run("collision", func(t *testing.T) {
		a := []qemu.Argument{
			qemu.UniqueArg("kernel", "vmlinuz"),
			qemu.UniqueArg("kernel", "bsd"),
		}
		_, err := qemu.BuildArgumentStrings(a)
		assert.Error(t, err)
	})
}
