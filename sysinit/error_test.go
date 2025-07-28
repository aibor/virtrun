// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit_test

import (
	"testing"

	"github.com/aibor/virtrun/sysinit"
	"github.com/stretchr/testify/assert"
)

func TestOptionalMountError_Is(t *testing.T) {
	tests := []struct {
		name   string
		other  error
		assert assert.BoolAssertionFunc
	}{
		{
			name:   "nil",
			assert: assert.False,
		},
		{
			name:   "same",
			other:  sysinit.OptionalMountError{assert.AnError},
			assert: assert.True,
		},
		{
			name:   "other",
			other:  assert.AnError,
			assert: assert.False,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sysinit.OptionalMountError{}
			tt.assert(t, err.Is(tt.other))
		})
	}
}

func TestOptionalMountError_Unwrap(t *testing.T) {
	err := sysinit.OptionalMountError{assert.AnError}
	assert.Equal(t, []error{assert.AnError}, err.Unwrap())
}
