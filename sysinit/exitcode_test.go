// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/aibor/virtrun/sysinit"
	"github.com/stretchr/testify/assert"
)

func TestExitCodeIdentifier_Sprint(t *testing.T) {
	e := sysinit.ExitCodeIdentifier("rc")

	assert.Equal(t, "rc: 42", e.Sprint(42))
}

func TestExitCodeIdentifier_ParseExitCode(t *testing.T) {
	tests := []struct {
		name        string
		exitCodeID  sysinit.ExitCodeIdentifier
		input       string
		expected    int
		assertFound assert.BoolAssertionFunc
	}{
		{
			name:        "empty input",
			exitCodeID:  "rc",
			assertFound: assert.False,
		},
		{
			name:        "matching input zero",
			exitCodeID:  "rc",
			input:       "rc: 0",
			assertFound: assert.True,
		},
		{
			name:        "matching input",
			exitCodeID:  "rc",
			input:       "rc: 42",
			expected:    42,
			assertFound: assert.True,
		},
		{
			name:        "matching input with trailing",
			exitCodeID:  "rc",
			input:       "rc: 42 whatever",
			expected:    42,
			assertFound: assert.True,
		},
		{
			name:        "matching input with leading",
			exitCodeID:  "rc",
			input:       "whatever rc: 42",
			expected:    42,
			assertFound: assert.True,
		},
		{
			name:        "with spaces",
			exitCodeID:  "exit code",
			input:       "exit code: 42",
			expected:    42,
			assertFound: assert.True,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, found := tt.exitCodeID.ParseExitCode(tt.input)
			tt.assertFound(t, found)

			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestExitCodeIdentifier_Printer(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedOut string
	}{
		{
			name:        "no error",
			expectedOut: "rc: 0\n",
		},
		{
			name:        "an error",
			err:         assert.AnError,
			expectedOut: "rc: -1\n",
		},
		{
			name:        "exit error",
			err:         sysinit.ExitError(42),
			expectedOut: "rc: 42\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buffer bytes.Buffer

			e := sysinit.ExitCodeIdentifier("rc")
			e.Printer(&buffer)(tt.err)

			assert.Equal(t, tt.expectedOut, buffer.String())
		})
	}
}

func TestExitCodeFrom(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "no error",
			expected: 0,
		},
		{
			name:     "an error",
			err:      assert.AnError,
			expected: -1,
		},
		{
			name:     "exit error",
			err:      sysinit.ExitError(42),
			expected: 42,
		},
		{
			name:     "wrapped exit error",
			err:      fmt.Errorf("test: %w", sysinit.ExitError(42)),
			expected: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := sysinit.ExitCodeFrom(tt.err)

			assert.Equal(t, tt.expected, actual)
		})
	}
}
