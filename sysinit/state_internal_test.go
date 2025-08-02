// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"bytes"
	"testing"

	"github.com/aibor/virtrun/internal/exitcode"
	"github.com/stretchr/testify/assert"
)

func TestState_Cleanup(t *testing.T) {
	calls := []int{}

	state := new(State)
	state.Cleanup(func() error {
		calls = append(calls, 1)
		return nil
	})

	state.Cleanup(func() error {
		calls = append(calls, 2)
		return assert.AnError
	})

	state.Cleanup(func() error {
		calls = append(calls, 3)
		return nil
	})

	errs := []error{}

	state.doCleanup(func(err error) {
		errs = append(errs, err)
	})

	assert.Equal(t, []int{3, 2, 1}, calls)

	if assert.Len(t, errs, 1, "expected errors") {
		assert.ErrorIs(t, errs[0], assert.AnError)
	}
}

func TestState_ExitCode(t *testing.T) {
	tests := []struct {
		name        string
		fn          func(*State)
		expectedOut string
	}{
		{
			name: "not set",
			fn:   func(_ *State) {},
		},
		{
			name:        "zero",
			fn:          func(s *State) { s.SetExitCode(0) },
			expectedOut: exitcode.Sprint(0) + "\n",
		},
		{
			name:        "non zero",
			fn:          func(s *State) { s.SetExitCode(269) },
			expectedOut: exitcode.Sprint(269) + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer

			state := new(State)
			tt.fn(state)
			state.printExitCode(&output)
			assert.Equal(t, tt.expectedOut, output.String())
		})
	}
}
