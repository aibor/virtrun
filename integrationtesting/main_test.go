// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

//go:build integration

package integrationtesting_test

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

var (
	KernelPath    = "/kernels/vmlinuz"
	KernelArch    = runtime.GOARCH
	KernelModules []string
	Verbose       bool
)

func TestMain(m *testing.M) {
	flag.StringVar(
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
	flag.Func(
		"kernel.module",
		"kernel module to add to guest. Flag may be used more than once.",
		func(s string) error {
			if s == "" {
				return errors.New("file path must not be empty")
			}

			path, err := filepath.Abs(s)
			if err != nil {
				return err
			}

			KernelModules = append(KernelModules, path)

			return nil
		},
	)
	flag.Parse()

	if !filepath.IsAbs(KernelPath) {
		fmt.Fprintf(os.Stderr, "KernelPath must be absolute: %v", KernelPath)
		os.Exit(1)
	}

	os.Exit(m.Run())
}
