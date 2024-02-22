// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package qemu

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommmandArgs(t *testing.T) {
	t.Run("yes-kvm", func(t *testing.T) {
		cmd := Command{}
		args := cmd.Args()
		assert.Contains(t, args, UniqueArg("enable-kvm"))
	})

	t.Run("no-kvm", func(t *testing.T) {
		cmd := Command{
			NoKVM: true,
		}
		args := cmd.Args()
		assert.NotContains(t, args, UniqueArg("enable-kvm"))
	})

	t.Run("yes-verbose", func(t *testing.T) {
		cmd := Command{
			Verbose: true,
		}
		args := cmd.Args()
		assert.NotContains(t, args[len(args)-1].Value(), "quiet")
	})

	t.Run("no-verbose", func(t *testing.T) {
		cmd := Command{}
		args := cmd.Args()
		assert.Contains(t, args[len(args)-1].Value(), "quiet")
	})

	t.Run("serial files virtio-mmio", func(t *testing.T) {
		cmd := Command{
			AdditionalConsoles: []string{
				"/output/file1",
				"/output/file2",
			},
			TransportType: TransportTypeMMIO,
		}

		expected := Arguments{
			ArgChardev("file,id=vcon1,path=/dev/fd/1"),
			ArgChardev("file,id=vcon3,path=/dev/fd/3"),
			ArgChardev("file,id=vcon4,path=/dev/fd/4"),
		}

		found := 0
		for _, a := range cmd.Args() {
			if a.Name() != "chardev" {
				continue
			}
			if assert.Less(t, found, len(expected), "expected serial files already consumed") {
				assert.Equal(t, expected[found], a)
			}
			found++
		}
		assert.Equal(t, len(expected), found, "all expected serial files should have been found")
	})

	t.Run("serial files isa-pci", func(t *testing.T) {
		cmd := Command{
			AdditionalConsoles: []string{
				"/output/file1",
				"/output/file2",
			},
			TransportType: TransportTypeISA,
		}

		expected := Arguments{
			ArgSerial("file:/dev/fd/1"),
			ArgSerial("file:/dev/fd/3"),
			ArgSerial("file:/dev/fd/4"),
		}

		found := 0
		for _, a := range cmd.Args() {
			if a.Name() != "serial" {
				continue
			}
			if assert.Less(t, found, len(expected), "expected serial files already consumed") {
				assert.Equal(t, expected[found], a)
			}
			found++
		}
		assert.Equal(t, len(expected), found, "all expected serial files should have been found")
	})

	t.Run("init args", func(t *testing.T) {
		cmd := Command{
			InitArgs: []string{
				"first",
				"second",
				"third",
			},
		}

		expected := " -- first second third"

		var appendValue string
		for _, a := range cmd.Args() {
			if a.Name() == "append" {
				appendValue = a.Value()
			}
		}

		require.NotEmpty(t, appendValue, "append value must be found")
		assert.Contains(t, appendValue, expected, "append value should contain init args")
	})
}
