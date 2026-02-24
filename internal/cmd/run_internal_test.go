// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"flag"
	"log"
	"testing"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
)

func TestHandleParseArgsError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
		expectedOut  string
	}{
		{
			name: "flag help",
			err:  flag.ErrHelp,
		},
		{
			name:         "parse args error",
			err:          &ParseArgsError{},
			expectedCode: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdErr bytes.Buffer

			log.SetOutput(&stdErr)
			log.SetFlags(0)

			actualExitCode := handleParseArgsError(tt.err)

			assert.Equal(t, tt.expectedCode, actualExitCode,
				"exit code should be as expected")
			assert.Equal(t, tt.expectedOut, stdErr.String(),
				"stderr output should be as expected")
		})
	}
}

func TestHandleRunError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
		expectedOut  string
	}{
		{
			name: "qemu command host error",
			err: &qemu.CommandError{
				Err:      assert.AnError,
				ExitCode: 42,
			},
			expectedCode: 42,
			expectedOut: "ERROR host: " +
				"assert.AnError general error for testing\n",
		},
		{
			name: "qemu command guest non-zero exit code error",
			err: &qemu.CommandError{
				Err:      qemu.ErrGuestNonZeroExitCode,
				Guest:    true,
				ExitCode: 43,
			},
			expectedCode: 43,
		},
		{
			name: "qemu command guest no exit code found error",
			err: &qemu.CommandError{
				Err:   qemu.ErrGuestNoExitCodeFound,
				Guest: true,
			},
			expectedCode: -1,
			expectedOut:  "ERROR guest: no exit code found\n",
		},
		{
			name: "qemu command guest oom error",
			err: &qemu.CommandError{
				Err:   qemu.ErrGuestOom,
				Guest: true,
			},
			expectedCode: -1,
			expectedOut:  "ERROR guest: system ran out of memory\n",
		},
		{
			name: "qemu command guest panic error",
			err: &qemu.CommandError{
				Err:   qemu.ErrGuestPanic,
				Guest: true,
			},
			expectedCode: -1,
			expectedOut:  "ERROR guest: system panicked\n",
		},
		{
			name:         "any error",
			err:          assert.AnError,
			expectedCode: -1,
			expectedOut:  "ERROR assert.AnError general error for testing\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdErr bytes.Buffer

			log.SetOutput(&stdErr)
			log.SetFlags(0)

			actualExitCode := handleRunError(tt.err)

			assert.Equal(t, tt.expectedCode, actualExitCode,
				"exit code should be as expected")
			assert.Equal(t, tt.expectedOut, stdErr.String(),
				"stderr output should be as expected")
		})
	}
}
