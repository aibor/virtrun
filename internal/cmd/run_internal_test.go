// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"flag"
	"testing"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
)

func TestHandleRunError(t *testing.T) {
	tests := []struct {
		name             string
		err              error
		expectedExitCode int
		expectedOutput   string
	}{
		{
			name: "no error",
		},
		{
			name: "flag help",
			err:  flag.ErrHelp,
		},
		{
			name:             "parse args error",
			err:              &ParseArgsError{},
			expectedExitCode: -1,
		},
		{
			name: "qemu command host error",
			err: &qemu.CommandError{
				Err:      assert.AnError,
				ExitCode: 42,
			},
			expectedExitCode: 42,
			expectedOutput: "Error [virtrun]: qemu host: " +
				"assert.AnError general error for testing\n",
		},
		{
			name: "qemu command guest non-zero exit code error",
			err: &qemu.CommandError{
				Err:      qemu.ErrGuestNonZeroExitCode,
				Guest:    true,
				ExitCode: 43,
			},
			expectedExitCode: 43,
		},
		{
			name: "qemu command guest no exit code found error",
			err: &qemu.CommandError{
				Err:   qemu.ErrGuestNoExitCodeFound,
				Guest: true,
			},
			expectedExitCode: -1,
			expectedOutput: "Error [virtrun]: qemu guest: " +
				"guest did not print init exit code\n",
		},
		{
			name: "qemu command guest oom error",
			err: &qemu.CommandError{
				Err:   qemu.ErrGuestOom,
				Guest: true,
			},
			expectedExitCode: -1,
			expectedOutput: "Error [virtrun]: qemu guest: " +
				"guest system ran out of memory\n",
		},
		{
			name: "qemu command guest panic error",
			err: &qemu.CommandError{
				Err:   qemu.ErrGuestPanic,
				Guest: true,
			},
			expectedExitCode: -1,
			expectedOutput: "Error [virtrun]: qemu guest: " +
				"guest system panicked\n",
		},
		{
			name:             "any error",
			err:              assert.AnError,
			expectedExitCode: -1,
			expectedOutput: "Error [virtrun]: " +
				"assert.AnError general error for testing\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdErr bytes.Buffer
			actualExitCode := handleRunError(tt.err, &stdErr)

			assert.Equal(t, tt.expectedExitCode, actualExitCode,
				"exit code should be as expected")
			assert.Equal(t, tt.expectedOutput, stdErr.String(),
				"stderr output should be as expected")
		})
	}
}
