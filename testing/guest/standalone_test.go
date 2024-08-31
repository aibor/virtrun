// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build standalone

package main_test

import (
	"testing"

	"github.com/aibor/virtrun/sysinit"
)

func TestMain(m *testing.M) {
	cfg := sysinit.DefaultConfig()
	cfg.ModulesDir = "/lib/modules"

	sysinit.RunTests(m, cfg)
}
