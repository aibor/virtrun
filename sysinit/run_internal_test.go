// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"bytes"
	"errors"
	"log"
	"testing"

	"github.com/aibor/virtrun/internal/exitcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.NotPanics(t, func() {
			run(nil, func(_ error) {}, nil)
		})
	})

	t.Run("exit handler", func(t *testing.T) {
		tests := []struct {
			name        string
			funcs       []Func
			expectedOut string
			expectedErr error
		}{
			{
				name:        "without error",
				expectedErr: nil,
			},
			{
				name: "with exit code",
				funcs: []Func{
					func(state *State) error {
						state.SetExitCode(42)
						return nil
					},
				},
				expectedErr: nil,
				expectedOut: exitcode.Sprint(42) + "\n",
			},
			{
				name: "with error",
				funcs: []Func{
					func(_ *State) error { return assert.AnError },
				},
				expectedErr: assert.AnError,
			},
			{
				name: "with exit code but error",
				funcs: []Func{
					func(state *State) error {
						state.SetExitCode(42)
						return nil
					},
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
				var called error

				var output bytes.Buffer

				exitHandler := func(err error) {
					require.NoError(t, called, "exit handler already called")
					called = err
				}

				run(&output, exitHandler, tt.funcs)

				require.ErrorIs(t, called, tt.expectedErr)
				assert.Equal(t, tt.expectedOut, output.String())
			})
		}
	})
}

func TestRunFuncs(t *testing.T) {
	tests := []struct {
		name           string
		funcs          []Func
		expectedErr    error
		assertExitCode assert.ValueAssertionFunc
	}{
		{
			name:           "none",
			assertExitCode: assert.Nil,
		},
		{
			name: "success",
			funcs: []Func{
				func(_ *State) error { return nil },
				func(_ *State) error { return nil },
			},
			assertExitCode: assert.Nil,
		},
		{
			name: "success with exit code",
			funcs: []Func{
				func(state *State) error {
					state.SetExitCode(42)
					return nil
				},
				func(_ *State) error { return nil },
			},
			assertExitCode: func(tt assert.TestingT, ec any, _ ...any) bool {
				expected := 42
				return assert.NotNil(tt, ec) && assert.Equal(t, &expected, ec)
			},
		},
		{
			name: "first fails",
			funcs: []Func{
				func(_ *State) error { return assert.AnError },
				func(_ *State) error { return errors.New("second") },
			},
			expectedErr:    assert.AnError,
			assertExitCode: assert.Nil,
		},
		{
			name: "second fails",
			funcs: []Func{
				func(_ *State) error { return nil },
				func(_ *State) error { return assert.AnError },
				func(_ *State) error { return errors.New("third") },
			},
			expectedErr:    assert.AnError,
			assertExitCode: assert.Nil,
		},
		{
			name: "panic without error",
			funcs: []Func{
				func(_ *State) error { panic(true) },
			},
			expectedErr:    ErrPanic,
			assertExitCode: assert.Nil,
		},
		{
			name: "panic with error",
			funcs: []Func{
				func(_ *State) error { panic(assert.AnError) },
			},
			expectedErr:    assert.AnError,
			assertExitCode: assert.Nil,
		},
		{
			name: "cleanup with error and error",
			funcs: []Func{
				func(state *State) error {
					state.Cleanup(func() error {
						return assert.AnError
					})

					return assert.AnError
				},
			},
			expectedErr:    assert.AnError,
			assertExitCode: assert.Nil,
		},
		{
			name: "cleanup with error",
			funcs: []Func{
				func(state *State) error {
					state.Cleanup(func() error {
						return assert.AnError
					})

					return nil
				},
			},
			assertExitCode: assert.Nil,
		},
		{
			name: "with exit code but cleanup with error",
			funcs: []Func{
				func(state *State) error {
					state.Cleanup(func() error {
						return assert.AnError
					})

					return nil
				},
				func(state *State) error {
					state.SetExitCode(42)
					return nil
				},
			},
			assertExitCode: func(tt assert.TestingT, ec any, _ ...any) bool {
				expected := 42
				return assert.NotNil(tt, ec) && assert.Equal(t, &expected, ec)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logOut bytes.Buffer

			log.SetFlags(0)
			log.SetOutput(&logOut)

			state := new(State)

			err := runFuncs(state, tt.funcs)
			require.ErrorIs(t, err, tt.expectedErr)
			tt.assertExitCode(t, state.exitCode)
		})
	}
}
