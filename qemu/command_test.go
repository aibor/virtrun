package qemu_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aibor/virtrun/qemu"
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
