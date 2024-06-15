// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package qemu_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStdoutProcessor(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		rc      int
		input   []string
		output  []string
		err     error
	}{
		{
			name: "panic",
			input: []string{
				"[    0.578502] Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000100",
				"[    0.579013] CPU: 0 PID: 76 Comm: init Not tainted 6.4.3-arch1-1 #1 13c144d261447e0acbf2632534d4009bddc4c3ab",
				"[    0.579512] Hardware name: QEMU Standard PC (Q35 + ICH9, 2009), BIOS Arch Linux 1.16.2-1-1 04/01/2014",
			},
			output: []string{""},
			err:    qemu.ErrGuestPanic,
		},
		{
			name: "oom",
			input: []string{
				//nolint:lll
				"[    0.378012] oom-kill:constraint=CONSTRAINT_NONE,nodemask=(null),cpuset=/,mems_allowed=0,global_oom,task_memcg=/,task=main,pid=116,uid=0",
				//nolint:lll
				"[    0.378083] Out of memory: Killed process 116 (main) total-vm:48156kB, anon-rss:43884kB, file-rss:4kB, shmem-rss:2924kB, UID:0 pgtables:140kB oom_score_adj:0",
			},
			output: []string{""},
			err:    qemu.ErrGuestOom,
		},
		{
			name:    "panic verbose",
			verbose: true,
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
			err: qemu.ErrGuestPanic,
		},
		{
			name: "rc",
			rc:   4,
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
			err: qemu.ErrGuestNoRCFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdOut := bytes.NewBufferString(strings.Join(tt.input, "\n"))
			stdOut := bytes.NewBuffer(make([]byte, 0, 512))

			rc, err := qemu.ParseStdout(cmdOut, stdOut, tt.verbose)

			if tt.err != nil {
				assert.ErrorIs(t, err, tt.err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.rc, rc)
		})
	}
}
