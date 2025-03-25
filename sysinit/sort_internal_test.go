// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortedMap(t *testing.T) {
	tests := []struct {
		name     string
		inputMap map[string]struct{}
		expected []string
	}{
		{
			name: "flat",
			inputMap: map[string]struct{}{
				"/dev":  {},
				"/sys":  {},
				"/proc": {},
				"/run":  {},
				"/tmp":  {},
			},
			expected: []string{
				"/dev",
				"/proc",
				"/run",
				"/sys",
				"/tmp",
			},
		},
		{
			name: "with sub dirs",
			inputMap: map[string]struct{}{
				"/dev":                {},
				"/sys":                {},
				"/proc":               {},
				"/run":                {},
				"/tmp":                {},
				"/sys/kernel/tracing": {},
				"/sys/fs/bpf":         {},
				"/dev/pts":            {},
			},
			expected: []string{
				"/dev",
				"/dev/pts",
				"/proc",
				"/run",
				"/sys",
				"/sys/fs/bpf",
				"/sys/kernel/tracing",
				"/tmp",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := []string{}
			for path := range sortedMap(tt.inputMap) {
				actual = append(actual, path)
			}

			assert.Equal(t, tt.expected, actual)
		})
	}
}
