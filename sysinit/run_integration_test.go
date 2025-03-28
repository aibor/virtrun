// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration_sysinit

package sysinit_test

import (
	"os"
	"testing"

	"github.com/aibor/virtrun/sysinit"
)

func TestMain(m *testing.M) {
	sysinit.Run(
		sysinit.ExitCodeID.Printer(os.Stdout),
		// Mount /tmp so gocoverdir works.
		sysinit.WithMountPoints(sysinit.MountPoints{
			"/dev":  {FSType: sysinit.FSTypeDevTmp},
			"/proc": {FSType: sysinit.FSTypeProc},
			"/tmp":  {FSType: sysinit.FSTypeTmp},
		}),
		func() error {
			if exitCode := m.Run(); exitCode != 0 {
				return sysinit.ExitError(exitCode)
			}

			return nil
		},
	)
}
