// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseModuleType(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		expected moduleType
	}{
		{
			name:     "empty",
			fileName: "",
			expected: moduleTypeUnknown,
		},
		{
			name:     "other",
			fileName: "other.gz",
			expected: moduleTypeUnknown,
		},
		{
			name:     "not extension",
			fileName: "zst.some",
			expected: moduleTypeUnknown,
		},
		{
			name:     "almost",
			fileName: "some_ko.gz",
			expected: moduleTypeUnknown,
		},
		{
			name:     "plain",
			fileName: "some.ko",
			expected: moduleTypePlain,
		},
		{
			name:     "gzip",
			fileName: "some.ko.gz",
			expected: moduleTypeGZIP,
		},
		{
			name:     "xz",
			fileName: "some.ko.xz",
			expected: moduleTypeXZ,
		},
		{
			name:     "zst",
			fileName: "some.ko.zst",
			expected: moduleTypeZSTD,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := parseModuleType(tt.fileName)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestFinitFlagsFor(t *testing.T) {
	tests := []struct {
		name       string
		moduleType moduleType
		expected   finitFlags
	}{
		{
			name:       "unknown",
			moduleType: moduleTypeUnknown,
			expected:   0,
		},
		{
			name:       "other",
			moduleType: "other",
			expected:   0,
		},
		{
			name:       "plain",
			moduleType: moduleTypePlain,
			expected:   0,
		},
		{
			name:       "gzip",
			moduleType: moduleTypeGZIP,
			expected:   finitFlagCompressedFile,
		},
		{
			name:       "xz",
			moduleType: moduleTypeXZ,
			expected:   finitFlagCompressedFile,
		},
		{
			name:       "zst",
			moduleType: moduleTypeZSTD,
			expected:   finitFlagCompressedFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := finitFlagsFor(tt.moduleType)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
