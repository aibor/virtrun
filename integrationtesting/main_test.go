// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

//go:build integration

package integrationtesting_test

import (
	"flag"
	"os"
	"runtime"
	"testing"

	"github.com/aibor/virtrun/internal/cmd"
)

var (
	KernelPath    = cmd.FilePath("/kernels/vmlinuz")
	KernelArch    = runtime.GOARCH
	KernelModules cmd.FilePathList
	Verbose       bool
)

func TestMain(m *testing.M) {
	flag.TextVar(
		&KernelPath,
		"kernel.path",
		KernelPath,
		"absolute path of the test kernel",
	)
	flag.StringVar(
		&KernelArch,
		"kernel.arch",
		KernelArch,
		"architecture of the kernel",
	)
	flag.BoolVar(
		&Verbose,
		"verbose",
		Verbose,
		"show complete guest output",
	)
	flag.Var(
		&KernelModules,
		"kernel.module",
		"kernel module to add to guest. Flag may be used more than once.",
	)
	flag.Parse()

	os.Exit(m.Run())
}
