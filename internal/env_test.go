// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package internal_test

import (
	"testing"

	"github.com/aibor/virtrun/internal"
	"github.com/stretchr/testify/assert"
)

func TestAddArgsFromEnv(t *testing.T) {
	tests := []struct {
		name   string
		env    string
		input  []string
		output []string
	}{
		{
			name:   "empty",
			env:    "",
			input:  []string{},
			output: []string{},
		},
		{
			name:   "only input, empty env",
			env:    "",
			input:  []string{"-kernel", "/boot/vmlinuz"},
			output: []string{"-kernel", "/boot/vmlinuz"},
		},
		{
			name:   "only env, empty input",
			env:    "-kernel /boot/vmlinuz",
			input:  []string{},
			output: []string{"-kernel", "/boot/vmlinuz"},
		},
		{
			name:   "both used",
			env:    "-kernel /boot/vmlinuz",
			input:  []string{"-verbose"},
			output: []string{"-kernel", "/boot/vmlinuz", "-verbose"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			varName := "VIRTRUN_ARGS"
			t.Setenv(varName, tt.env)
			assert.Equal(t, tt.output, internal.PrependEnvArgs(tt.input))
		})
	}
}
