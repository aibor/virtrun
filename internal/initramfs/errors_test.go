// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package initramfs_test

import (
	"errors"
	"testing"

	"github.com/aibor/virtrun/internal/initramfs"
	"github.com/stretchr/testify/assert"
)

func TestArchiveErrorIs(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
		assert assert.BoolAssertionFunc
	}{
		{
			name:   "empty",
			err:    &initramfs.ArchiveError{},
			target: &initramfs.ArchiveError{},
			assert: assert.True,
		},
		{
			name: "same op",
			err: &initramfs.ArchiveError{
				Op: "write",
			},
			target: &initramfs.ArchiveError{
				Op: "write",
			},
			assert: assert.True,
		},
		{
			name: "different op",
			err: &initramfs.ArchiveError{
				Op: "write",
			},
			target: &initramfs.ArchiveError{
				Op: "reaf",
			},
			assert: assert.False,
		},
		{
			name: "other error",
			err:  errors.New("fail"),
			target: &initramfs.ArchiveError{
				Op: "reaf",
			},
			assert: assert.False,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assert(t, errors.Is(tt.err, tt.target))
		})
	}
}
