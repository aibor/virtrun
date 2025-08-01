// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"testing"

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
		return nil
	})

	state.doCleanup()

	assert.Equal(t, []int{2, 1}, calls)
}
