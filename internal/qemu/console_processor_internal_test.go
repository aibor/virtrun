package qemu

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScrubCR(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectedErr error
	}{
		{
			name: "empty",
		},
		{
			name:     "crlf only",
			input:    "\r\n",
			expected: "\n",
		},
		{
			name:     "lf only",
			input:    "\n",
			expected: "\n",
		},
		{
			name:     "with crlf",
			input:    "some first\r\nand second\r\nand third line",
			expected: "some first\nand second\nand third line\n",
		},
		{
			name:     "with lf",
			input:    "some first\nand second\nand third line",
			expected: "some first\nand second\nand third line\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer

			processor, writePipe, err := scrubCR(&output)
			require.NoError(t, err)

			go func() {
				_, _ = io.Copy(writePipe, bytes.NewBufferString(tt.input))
				_ = writePipe.Close()
			}()

			err = processor()
			require.ErrorIs(t, err, tt.expectedErr)

			assert.Equal(t, tt.expected, output.String())
		})
	}
}

func TestParseExitCode(t *testing.T) {
	tests := []struct {
		name        string
		verbose     bool
		input       []string
		exitCode    int
		expected    []string
		expectedErr error
	}{
		{
			name: "oom",
			input: []string{
				//nolint:lll
				"[    0.378012] oom-kill:constraint=CONSTRAINT_NONE,nodemask=(null),cpuset=/,mems_allowed=0,global_oom,task_memcg=/,task=main,pid=116,uid=0",
				//nolint:lll
				"[    0.378083] Out of memory: Killed process 116 (main) total-vm:48156kB, anon-rss:43884kB, file-rss:4kB, shmem-rss:2924kB, UID:0 pgtables:140kB oom_score_adj:0",
			},
			expected: []string{
				//nolint:lll
				"[    0.378012] oom-kill:constraint=CONSTRAINT_NONE,nodemask=(null),cpuset=/,mems_allowed=0,global_oom,task_memcg=/,task=main,pid=116,uid=0",
				//nolint:lll
				"[    0.378083] Out of memory: Killed process 116 (main) total-vm:48156kB, anon-rss:43884kB, file-rss:4kB, shmem-rss:2924kB, UID:0 pgtables:140kB oom_score_adj:0",
				"",
			},
			expectedErr: ErrGuestOom,
		},
		{
			name: "panic",
			input: []string{
				"[    0.578502] Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000100",
				"[    0.579013] CPU: 0 PID: 76 Comm: init Not tainted 6.4.3-arch1-1 #1 13c144d261447e0acbf2632534d4009bddc4c3ab",
				"[    0.579512] Hardware name: QEMU Standard PC (Q35 + ICH9, 2009), BIOS Arch Linux 1.16.2-1-1 04/01/2014",
			},
			expected: []string{
				"[    0.578502] Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000100",
				"[    0.579013] CPU: 0 PID: 76 Comm: init Not tainted 6.4.3-arch1-1 #1 13c144d261447e0acbf2632534d4009bddc4c3ab",
				"[    0.579512] Hardware name: QEMU Standard PC (Q35 + ICH9, 2009), BIOS Arch Linux 1.16.2-1-1 04/01/2014",
				"",
			},
			expectedErr: ErrGuestPanic,
		},
		{
			name:     "zero rc",
			exitCode: 0,
			input: []string{
				"something out",
				fmt.Sprintf(RCFmt, 0),
				"more after",
			},
			expected: []string{
				"something out",
				"",
			},
		},
		{
			name:     "zero rc verbose",
			verbose:  true,
			exitCode: 0,
			input: []string{
				"something out",
				fmt.Sprintf(RCFmt, 0),
				"more after",
			},
			expected: []string{
				"something out",
				fmt.Sprintf(RCFmt, 0),
				"more after",
				"",
			},
		},
		{
			name:     "non zero rc",
			exitCode: 4,
			input: []string{
				"something out",
				fmt.Sprintf(RCFmt, 4),
				"more after",
			},
			expected: []string{
				"something out",
				"",
			},
			expectedErr: ErrGuestNonZeroExitCode,
		},
		{
			name:     "non zero rc verbose",
			verbose:  true,
			exitCode: 4,
			input: []string{
				"something out",
				fmt.Sprintf(RCFmt, 4),
				"more after",
			},
			expected: []string{
				"something out",
				fmt.Sprintf(RCFmt, 4),
				"more after",
				"",
			},
			expectedErr: ErrGuestNonZeroExitCode,
		},
		{
			name: "no rc",
			input: []string{
				"something out",
				"more out",
			},
			expected: []string{
				"something out",
				"more out",
				"",
			},
			expectedErr: ErrGuestNoExitCodeFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdOut bytes.Buffer

			cmdOut := bytes.NewBufferString(strings.Join(tt.input, "\n"))

			err := parseStdout(&stdOut, cmdOut, tt.verbose)()
			require.ErrorIs(t, err, tt.expectedErr)

			var (
				cmdErr *CommandError
				rc     int
			)

			if errors.As(err, &cmdErr) {
				rc = cmdErr.ExitCode
			}

			assert.Equal(t, tt.exitCode, rc)
			assert.Equal(t, tt.expected, strings.Split(stdOut.String(), "\n"))
		})
	}
}
