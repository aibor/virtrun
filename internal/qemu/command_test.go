package qemu_test

import (
	"testing"

	"github.com/aibor/pidonetest/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommmandConsoleDeviceName(t *testing.T) {
	tests := []struct {
		id        uint8
		transport qemu.TransportType
		console   string
	}{
		{
			id:        5,
			transport: qemu.TransportTypeISA,
			console:   "ttyS5",
		},
		{
			id:        3,
			transport: qemu.TransportTypePCI,
			console:   "hvc3",
		},
		{
			id:        1,
			transport: qemu.TransportTypeMMIO,
			console:   "hvc1",
		},
	}
	for _, tt := range tests {
		c := qemu.Command{
			TransportType: tt.transport,
		}
		assert.Equal(t, tt.console, c.ConsoleDeviceName(tt.id))
	}
}

func TestCommmandAddExtraFile(t *testing.T) {
	c := qemu.Command{}
	d1 := c.AddExtraFile("test")
	d2 := c.AddExtraFile("real")
	assert.Equal(t, "ttyS1", d1)
	assert.Equal(t, "ttyS2", d2)
	assert.Equal(t, []string{"test", "real"}, c.ExtraFiles)
}

func TestCommmandArgs(t *testing.T) {
	t.Run("yes-kvm", func(t *testing.T) {
		q := qemu.Command{}
		args := q.Args()
		assert.Contains(t, args, qemu.UniqueArg("enable-kvm"))
	})

	t.Run("no-kvm", func(t *testing.T) {
		q := qemu.Command{
			NoKVM: true,
		}
		args := q.Args()
		assert.NotContains(t, args, qemu.UniqueArg("enable-kvm"))
	})

	t.Run("yes-verbose", func(t *testing.T) {
		q := qemu.Command{
			Verbose: true,
		}
		args := q.Args()
		assert.NotContains(t, args[len(args)-1].Value, "loglevel=0")
	})

	t.Run("no-verbose", func(t *testing.T) {
		q := qemu.Command{}
		args := q.Args()
		assert.Contains(t, args[len(args)-1].Value, "loglevel=0")
	})

	t.Run("serial files virtio-mmio", func(t *testing.T) {
		q := qemu.Command{
			ExtraFiles: []string{
				"/output/file1",
				"/output/file2",
			},
			TransportType: qemu.TransportTypeMMIO,
		}

		expected := qemu.Arguments{
			qemu.ArgChardev("file,id=vcon1,path=/dev/fd/1"),
			qemu.ArgChardev("file,id=vcon3,path=/dev/fd/3"),
			qemu.ArgChardev("file,id=vcon4,path=/dev/fd/4"),
		}

		found := 0
		for _, a := range q.Args() {
			if a.Name != "chardev" {
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
		q := qemu.Command{
			ExtraFiles: []string{
				"/output/file1",
				"/output/file2",
			},
			TransportType: qemu.TransportTypeISA,
		}

		expected := qemu.Arguments{
			qemu.ArgSerial("file:/dev/fd/1"),
			qemu.ArgSerial("file:/dev/fd/3"),
			qemu.ArgSerial("file:/dev/fd/4"),
		}

		found := 0
		for _, a := range q.Args() {
			if a.Name != "serial" {
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
		q := qemu.Command{
			InitArgs: []string{
				"first",
				"second",
				"third",
			},
		}

		expected := " -- first second third"

		var appendValue string
		for _, a := range q.Args() {
			if a.Name == "append" {
				appendValue = a.Value
			}
		}

		require.NotEmpty(t, appendValue, "append value must be found")
		assert.Contains(t, appendValue, expected, "append value should contain init args")
	})
}
