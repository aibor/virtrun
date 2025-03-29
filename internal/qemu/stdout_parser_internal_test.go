// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStdoutParser_Process(t *testing.T) {
	exitCodeFmt := "exit code: %d"

	tests := []struct {
		name                string
		verbose             bool
		input               []string
		expected            []string
		expectedExitCode    int
		assertExitCodeFound assert.BoolAssertionFunc
	}{
		{
			name: "oom",
			//nolint:lll
			input: []string{
				"[    0.378012] oom-kill:constraint=CONSTRAINT_NONE,nodemask=(null),cpuset=/,mems_allowed=0,global_oom,task_memcg=/,task=main,pid=116,uid=0",
				"[    0.378083] Out of memory: Killed process 116 (main) total-vm:48156kB, anon-rss:43884kB, file-rss:4kB, shmem-rss:2924kB, UID:0 pgtables:140kB oom_score_adj:0",
			},
			//nolint:lll
			expected: []string{
				"[    0.378012] oom-kill:constraint=CONSTRAINT_NONE,nodemask=(null),cpuset=/,mems_allowed=0,global_oom,task_memcg=/,task=main,pid=116,uid=0",
				"[    0.378083] Out of memory: Killed process 116 (main) total-vm:48156kB, anon-rss:43884kB, file-rss:4kB, shmem-rss:2924kB, UID:0 pgtables:140kB oom_score_adj:0",
			},
			assertExitCodeFound: assert.False,
		},
		{
			name: "panic",
			//nolint:lll
			input: []string{
				"[    0.578502] Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000100",
				"[    0.579013] CPU: 0 PID: 76 Comm: init Not tainted 6.4.3-arch1-1 #1 13c144d261447e0acbf2632534d4009bddc4c3ab",
				"[    0.579512] Hardware name: QEMU Standard PC (Q35 + ICH9, 2009), BIOS Arch Linux 1.16.2-1-1 04/01/2014",
			},
			//nolint:lll
			expected: []string{
				"[    0.578502] Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000100",
				"[    0.579013] CPU: 0 PID: 76 Comm: init Not tainted 6.4.3-arch1-1 #1 13c144d261447e0acbf2632534d4009bddc4c3ab",
				"[    0.579512] Hardware name: QEMU Standard PC (Q35 + ICH9, 2009), BIOS Arch Linux 1.16.2-1-1 04/01/2014",
			},
			assertExitCodeFound: assert.False,
		},
		{
			name: "zero exit code",
			input: []string{
				"something out",
				fmt.Sprintf(exitCodeFmt, 0),
				"more after",
			},
			expected: []string{
				"something out",
			},
			expectedExitCode:    0,
			assertExitCodeFound: assert.True,
		},
		{
			name:    "zero exit code verbose",
			verbose: true,
			input: []string{
				"something out",
				fmt.Sprintf(exitCodeFmt, 0),
				"more after",
			},
			expected: []string{
				"something out",
				fmt.Sprintf(exitCodeFmt, 0),
				"more after",
			},
			expectedExitCode:    0,
			assertExitCodeFound: assert.True,
		},
		{
			name: "non zero exit code",
			input: []string{
				"something out",
				fmt.Sprintf(exitCodeFmt, 4),
				"more after",
			},
			expected: []string{
				"something out",
			},
			expectedExitCode:    4,
			assertExitCodeFound: assert.True,
		},
		{
			name:    "non zero exit code verbose",
			verbose: true,
			input: []string{
				"something out",
				fmt.Sprintf(exitCodeFmt, 4),
				"more after",
			},
			expected: []string{
				"something out",
				fmt.Sprintf(exitCodeFmt, 4),
				"more after",
			},
			expectedExitCode:    4,
			assertExitCodeFound: assert.True,
		},
		{
			name: "no exit code",
			input: []string{
				"something out",
				"more out",
			},
			expected: []string{
				"something out",
				"more out",
			},
			assertExitCodeFound: assert.False,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actual []string

			stdoutParser := stdoutParser{
				Verbose: tt.verbose,
				ExitCodeParser: func(s string) (int, bool) {
					if d, found := strings.CutPrefix(s, "exit code: "); found {
						i, err := strconv.Atoi(d)
						return i, err == nil
					}

					return 0, false
				},
			}

			for _, line := range tt.input {
				out := stdoutParser.Parse([]byte(line))
				if out != nil {
					actual = append(actual, string(out))
				}
			}

			tt.assertExitCodeFound(t, stdoutParser.exitCodeFound, "exit code found")
			assert.Equal(t, tt.expectedExitCode, stdoutParser.exitCode, "exit code")
			assert.Equal(t, tt.expected, actual, "output")
		})
	}
}
