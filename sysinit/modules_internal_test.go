// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasFinitCompressionExtension(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		expected bool
	}{
		{
			name:     "empty",
			fileName: "",
			expected: false,
		},
		{
			name:     "other",
			fileName: "other",
			expected: false,
		},
		{
			name:     "not extension",
			fileName: "zst.some",
			expected: false,
		},
		{
			name:     "gzip",
			fileName: "some.gz",
			expected: true,
		},
		{
			name:     "xz",
			fileName: "some.xz",
			expected: true,
		},
		{
			name:     "zst",
			fileName: "some.zst",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := hasFinitCompressionExtension(tt.fileName)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
