// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration_sysinit

package sysinit_test

import (
	"os"
	"testing"

	"github.com/aibor/virtrun/sysinit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSymlinks(t *testing.T) {
	tests := []struct {
		name        string
		symlinks    sysinit.Symlinks
		expectedErr error
	}{
		{
			name: "empty set",
		},
		{
			name: "invalid link",
			symlinks: sysinit.Symlinks{
				"": "/tmp",
			},
			expectedErr: os.ErrNotExist,
		},
		{
			name: "valid symlinks",
			symlinks: sysinit.Symlinks{
				"/dev/core":   "/proc/kcore",
				"/dev/fd":     "/proc/self/fd/",
				"/dev/rtc":    "rtc0",
				"/dev/stdin":  "/proc/self/fd/0",
				"/dev/stdout": "/proc/self/fd/1",
				"/dev/stderr": "/proc/self/fd/2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				for link := range tt.symlinks {
					err := os.Remove(link)
					if err != nil && tt.expectedErr == nil {
						t.Logf("Failed to remove symlink %s: %v", link, err)
					}
				}
			})

			err := sysinit.CreateSymlinks(tt.symlinks)
			require.ErrorIs(t, err, tt.expectedErr)

			if tt.expectedErr != nil {
				return
			}

			for link, expectedTarget := range tt.symlinks {
				target, err := os.Readlink(link)
				if assert.NoError(t, err, "link should be readable") {
					assert.Equal(t, expectedTarget, target,
						"link target should be as expected")
				}
			}
		})
	}
}
