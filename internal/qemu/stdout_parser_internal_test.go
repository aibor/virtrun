// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStdoutParser(t *testing.T) {
	exitCodeFmt := "exit code: %d"

	exitCode := func(e int) string {
		return fmt.Sprintf(exitCodeFmt, e)
	}

	exitCodeScanner := func(line []byte) (int, bool) {
		var d int

		_, err := fmt.Sscanf(string(line), exitCodeFmt, &d)

		return d, err == nil
	}

	tests := []struct {
		name            string
		verbose         bool
		input           []string
		expectedOut     []string
		expectedCode    int
		assertCodeFound assert.BoolAssertionFunc
		expectedErr     error
	}{
		{
			name: "oom",
			//nolint:lll
			input: []string{
				"[    0.378012] oom-kill:constraint=CONSTRAINT_NONE,nodemask=(null),cpuset=/,mems_allowed=0,global_oom,task_memcg=/,task=main,pid=116,uid=0\n",
				"[    0.378083] Out of memory: Killed process 116 (main) total-vm:48156kB, anon-rss:43884kB, file-rss:4kB, shmem-rss:2924kB, UID:0 pgtables:140kB oom_score_adj:0\n",
			},
			//nolint:lll
			expectedOut: []string{
				"[    0.378012] oom-kill:constraint=CONSTRAINT_NONE,nodemask=(null),cpuset=/,mems_allowed=0,global_oom,task_memcg=/,task=main,pid=116,uid=0\n",
				"[    0.378083] Out of memory: Killed process 116 (main) total-vm:48156kB, anon-rss:43884kB, file-rss:4kB, shmem-rss:2924kB, UID:0 pgtables:140kB oom_score_adj:0\n",
			},
			assertCodeFound: assert.False,
			expectedErr:     ErrGuestOom,
		},
		{
			name: "panic",
			//nolint:lll
			input: []string{
				"[    0.578502] Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000100\n",
				"[    0.579013] CPU: 0 PID: 76 Comm: init Not tainted 6.4.3-arch1-1 #1 13c144d261447e0acbf2632534d4009bddc4c3ab\n",
				"[    0.579512] Hardware name: QEMU Standard PC (Q35 + ICH9, 2009), BIOS Arch Linux 1.16.2-1-1 04/01/2014\n",
			},
			//nolint:lll
			expectedOut: []string{
				"[    0.578502] Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000100\n",
				"[    0.579013] CPU: 0 PID: 76 Comm: init Not tainted 6.4.3-arch1-1 #1 13c144d261447e0acbf2632534d4009bddc4c3ab\n",
				"[    0.579512] Hardware name: QEMU Standard PC (Q35 + ICH9, 2009), BIOS Arch Linux 1.16.2-1-1 04/01/2014\n",
			},
			assertCodeFound: assert.False,
			expectedErr:     ErrGuestPanic,
		},
		{
			name: "zero exit code",
			input: []string{
				"something out\n",
				exitCode(0) + "\n",
				"more after\n",
			},
			expectedOut: []string{
				"something out\n",
			},
			assertCodeFound: assert.True,
		},
		{
			name:    "zero exit code verbose",
			verbose: true,
			input: []string{
				"something out\n",
				exitCode(0) + "\n",
				"more after\n",
			},
			expectedOut: []string{
				"something out\n",
				exitCode(0) + "\n",
				"more after\n",
			},
			assertCodeFound: assert.True,
		},
		{
			name: "non zero exit code",
			input: []string{
				"something out\n",
				exitCode(4) + "\n",
				"more after\n",
			},
			expectedOut: []string{
				"something out\n",
			},
			expectedCode:    4,
			assertCodeFound: assert.True,
			expectedErr:     ErrGuestNonZeroExitCode,
		},
		{
			name:    "non zero exit code verbose",
			verbose: true,
			input: []string{
				"something out\n",
				exitCode(4) + "\n",
				"more after\n",
			},
			expectedOut: []string{
				"something out\n",
				exitCode(4) + "\n",
				"more after\n",
			},
			expectedCode:    4,
			assertCodeFound: assert.True,
			expectedErr:     ErrGuestNonZeroExitCode,
		},
		{
			name: "no exit code",
			input: []string{
				"something out\n",
				"more out\n",
			},
			expectedOut: []string{
				"something out\n",
				"more out\n",
			},
			assertCodeFound: assert.False,
			expectedErr:     ErrGuestNoExitCodeFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer

			input := strings.NewReader(strings.Join(tt.input, ""))

			expectedBytes := int64(input.Len())

			parser := stdoutParser{
				ExitCodeParser: exitCodeScanner,
				Verbose:        tt.verbose,
			}

			actualBytes, err := parser.Copy(&output, input)
			require.NoError(t, err)

			assert.Equal(t, expectedBytes, actualBytes, "bytes read")

			actualOut := slices.Collect(strings.Lines(output.String()))
			assert.Equal(t, tt.expectedOut, actualOut, "output")

			require.ErrorIs(t, parser.err, tt.expectedErr, "parser error")
			assert.Equal(t, tt.expectedCode, parser.exitCode, "exit code")
		})
	}
}
