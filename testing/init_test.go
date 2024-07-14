// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

//go:build integration

package integration_test

import (
	"flag"

	"github.com/aibor/virtrun/internal"
)

//nolint:gochecknoglobals
var (
	KernelPath    = internal.FilePath("/kernels/vmlinuz")
	KernelArch    = internal.ArchNative
	KernelModules internal.FilePathList
	Verbose       bool
)

//nolint:gochecknoinits
func init() {
	flag.TextVar(
		&KernelPath,
		"kernel.path",
		KernelPath,
		"absolute path of the test kernel",
	)
	flag.TextVar(
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
}
