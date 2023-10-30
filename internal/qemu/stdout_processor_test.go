package qemu_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aibor/pidonetest/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStdoutProcessor(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		found   bool
		rc      int
		input   []string
		output  []string
	}{
		{
			name: "panic",
			rc:   126,
			input: []string{
				"[    0.578502] Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000100",
				"[    0.579013] CPU: 0 PID: 76 Comm: init Not tainted 6.4.3-arch1-1 #1 13c144d261447e0acbf2632534d4009bddc4c3ab",
				"[    0.579512] Hardware name: QEMU Standard PC (Q35 + ICH9, 2009), BIOS Arch Linux 1.16.2-1-1 04/01/2014",
			},
			output: []string{""},
		},
		{
			name:    "panic verbose",
			verbose: true,
			rc:      126,
			input: []string{
				"[    0.578502] Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000100",
				"[    0.579013] CPU: 0 PID: 76 Comm: init Not tainted 6.4.3-arch1-1 #1 13c144d261447e0acbf2632534d4009bddc4c3ab",
				"[    0.579512] Hardware name: QEMU Standard PC (Q35 + ICH9, 2009), BIOS Arch Linux 1.16.2-1-1 04/01/2014",
			},
			output: []string{
				"[    0.578502] Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000100",
				"[    0.579013] CPU: 0 PID: 76 Comm: init Not tainted 6.4.3-arch1-1 #1 13c144d261447e0acbf2632534d4009bddc4c3ab",
				"[    0.579512] Hardware name: QEMU Standard PC (Q35 + ICH9, 2009), BIOS Arch Linux 1.16.2-1-1 04/01/2014",
				"",
			},
		},
		{
			name:  "rc",
			found: true,
			rc:    4,
			input: []string{
				"something out",
				"more out",
				fmt.Sprintf(qemu.RCFmt, 4),
			},
			output: []string{
				"something out",
				"more out",
				"",
			},
		},
		{
			name: "no rc",
			input: []string{
				"something out",
				"more out",
			},
			output: []string{
				"something out",
				"more out",
				"",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cmdOut := bytes.NewBuffer([]byte(strings.Join(tt.input, "\n")))
			stdOut := bytes.NewBuffer(make([]byte, 0, 512))

			rcParser := qemu.NewStdoutProcessor(stdOut, tt.verbose)
			done := make(chan struct{})
			go func() {
				defer close(done)
				require.NoError(t, rcParser.Run())
			}()
			_, err := io.Copy(rcParser, cmdOut)
			require.NoError(t, err)
			require.NoError(t, rcParser.Close())

			select {
			case <-done:
				assert.Equal(t, tt.rc, rcParser.RC)
			case <-time.After(100 * time.Millisecond):
				assert.Fail(t, "rcParser did not return in time")
			}
		})
	}
}
