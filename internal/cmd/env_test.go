// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd_test

import (
	"testing"
	"testing/fstest"

	"github.com/aibor/virtrun/internal/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvArgs(t *testing.T) {
	tests := []struct {
		name   string
		env    string
		output []string
	}{
		{
			name:   "empty",
			env:    "",
			output: []string{},
		},
		{
			name:   "multiple args",
			env:    "-kernel /boot/vmlinuz",
			output: []string{"-kernel", "/boot/vmlinuz"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			varName := "VIRTRUN_ARGS"
			t.Setenv(varName, tt.env)
			assert.Equal(t, tt.output, cmd.EnvArgs())
		})
	}
}

func TestLocalConfigArgs(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		env      map[string]string
		expected []string
	}{
		{
			name:     "empty",
			content:  "",
			expected: []string{},
		},
		{
			name:     "single line",
			content:  "-arg1=3\n-arg2=4 5",
			expected: []string{"-arg1=3", "-arg2=4 5"},
		},
		{
			name:     "multiple lines",
			content:  "-arg1\n3\n-arg2\n4\n",
			expected: []string{"-arg1", "3", "-arg2", "4"},
		},
		{
			name:     "with env vars",
			content:  "-arg1=${VAR1}\n-arg2=$VAR2--\n-arg3=${VAR3}/more\n",
			env:      map[string]string{"VAR1": "42", "VAR2": "__"},
			expected: []string{"-arg1=42", "-arg2=__--", "-arg3=/more"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFS := fstest.MapFS{
				"conf": &fstest.MapFile{
					Data: []byte(tt.content),
				},
			}

			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			content, err := cmd.LocalConfigArgs(testFS, "conf")
			require.NoError(t, err)

			assert.Equal(t, tt.expected, content)
		})
	}
}
