package internal_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aibor/go-pidonetest/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputConsume(t *testing.T) {
	tests := []struct {
		name   string
		rc     *internal.RCValue
		input  []string
		output []string
	}{
		{
			name: "panic",
			rc:   &internal.RCValue{Found: false, RC: 126},
			input: []string{
				"[    0.578502] Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000100",
				"[    0.579013] CPU: 0 PID: 76 Comm: init Not tainted 6.4.3-arch1-1 #1 13c144d261447e0acbf2632534d4009bddc4c3ab",
				"[    0.579512] Hardware name: QEMU Standard PC (Q35 + ICH9, 2009), BIOS Arch Linux 1.16.2-1-1 04/01/2014",
			},
			output: []string{""},
		},
		{
			name: "rc",
			rc:   &internal.RCValue{Found: true, RC: 4},
			input: []string{
				"something out",
				"more out",
				fmt.Sprintf(internal.RCFmt, 4),
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
			cmdErr := bytes.NewBuffer([]byte{})
			stdErr := bytes.NewBuffer(make([]byte, 0, 512))

			wg, rcStream, err := internal.Consume(&internal.Output{
				OutReader: cmdOut,
				ErrReader: cmdErr,
				OutWriter: stdOut,
				ErrWriter: stdErr,
			})
			require.NoError(t, err, "Consume")

			assert.Eventually(t,
				func() bool { wg.Wait(); return true },
				50*time.Millisecond,
				10*time.Millisecond,
				"wait waitgroup")

			if assert.NoError(t, err, "ReadAll") {
				assert.Equal(t, tt.output, strings.Split(stdOut.String(), "\n"), "stdout")
			}

			if tt.rc != nil {
				if assert.Len(t, rcStream, 1, "channel has value") {
					assert.Equal(t, *tt.rc, <-rcStream, "panic return code")
				}
			}
		})
	}
}
