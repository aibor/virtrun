// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.NotPanics(t, func() {
			run(func(_ error) {}, nil)
		})
	})

	t.Run("exit handler", func(t *testing.T) {
		tests := []struct {
			name        string
			funcs       []Func
			expectedErr error
		}{
			{
				name:        "without error",
				expectedErr: nil,
			},
			{
				name: "with error",
				funcs: []Func{
					func(_ *State) error { return assert.AnError },
				},
				expectedErr: assert.AnError,
			},
			{
				name: "with panic",
				funcs: []Func{
					func(_ *State) error { panic(assert.AnError) },
				},
				expectedErr: assert.AnError,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var calledWith error

				exitHandler := func(err error) {
					require.NoError(
						t,
						calledWith,
						"exit handler already called",
					)

					calledWith = err
				}

				run(exitHandler, tt.funcs)

				require.ErrorIs(t, calledWith, tt.expectedErr)
			})
		}
	})
}

func TestRunFuncs(t *testing.T) {
	tests := []struct {
		name        string
		funcs       []Func
		expectedErr error
	}{
		{
			name: "none",
		},
		{
			name: "success",
			funcs: []Func{
				func(_ *State) error { return nil },
				func(_ *State) error { return nil },
			},
		},
		{
			name: "first fails",
			funcs: []Func{
				func(_ *State) error { return assert.AnError },
				func(_ *State) error { return errors.New("second") },
			},
			expectedErr: assert.AnError,
		},
		{
			name: "second fails",
			funcs: []Func{
				func(_ *State) error { return nil },
				func(_ *State) error { return assert.AnError },
				func(_ *State) error { return errors.New("third") },
			},
			expectedErr: assert.AnError,
		},
		{
			name:        "panic without error",
			funcs:       []Func{func(_ *State) error { panic(true) }},
			expectedErr: ErrPanic,
		},
		{
			name:        "panic with error",
			funcs:       []Func{func(_ *State) error { panic(assert.AnError) }},
			expectedErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runFuncs(tt.funcs)
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}
