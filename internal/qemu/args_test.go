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
	tests := []struct {
		name         string
		args         []qemu.Argument
		expect       []string
		requireError require.ErrorAssertionFunc
		assert       assert.ComparisonAssertionFunc
	}{
		{
			name: "builds",
			args: []qemu.Argument{
				qemu.UniqueArg("kernel", "vmlinuz"),
				qemu.UniqueArg("initrd", "boot"),
				qemu.UniqueArg("yes"),
				qemu.RepeatableArg("more", "a"),
				qemu.RepeatableArg("more", "b"),
			},
			expect: []string{
				"-kernel", "vmlinuz",
				"-initrd", "boot",
				"-yes",
				"-more", "a",
				"-more", "b",
			},
			requireError: require.NoError,
		},
		{
			name: "collision",
			args: []qemu.Argument{
				qemu.UniqueArg("kernel", "vmlinuz"),
				qemu.UniqueArg("kernel", "bsd"),
			},
			requireError: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := qemu.BuildArgumentStrings(tt.args)
			tt.requireError(t, err)
			assert.Equal(t, tt.expect, actual)
		})
	}
}
