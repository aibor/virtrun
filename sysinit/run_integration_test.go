// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration_sysinit

package sysinit_test

import (
	"testing"

	"github.com/aibor/virtrun/sysinit"
)

func TestMain(m *testing.M) {
	sysinit.Run(
		func(state *sysinit.State) error {
			state.SetExitCode(m.Run())
			return nil
		},
	)
}
