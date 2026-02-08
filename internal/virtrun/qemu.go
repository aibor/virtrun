// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"path/filepath"
	"strings"

	"github.com/aibor/virtrun/internal/qemu"
)

// RewriteGoTestFlagsPath processes file related go test flags so their file
// path are correct for use in the guest system.
//
// It is required that the flags are prefixed with "test" and value is
// separated form the flag by "=". This is the format the "go test" tool
// invokes the test binary with.
//
// Each file path is replaced with a path to a serial console. The modified args
// are returned along with a list of the host file paths.
func RewriteGoTestFlagsPath(spec *qemu.CommandSpec) {
	const splitNum = 2

	outputDir := ""

	for idx, posArg := range spec.InitArgs {
		splits := strings.SplitN(posArg, "=", splitNum)
		switch splits[0] {
		case "-test.outputdir":
			outputDir = splits[1]
			fallthrough
		case "-test.gocoverdir":
			splits[1] = "/tmp"
		default:
			continue
		}

		spec.InitArgs[idx] = strings.Join(splits, "=")
	}

	// Only coverprofile has a relative path to the test pwd and can be
	// replaced immediately. All other profile files are relative to the actual
	// test running and need to be prefixed with -test.outputdir. So, collect
	// them and process them afterwards when "outputdir" is found.
	for idx, posArg := range spec.InitArgs {
		splits := strings.SplitN(posArg, "=", splitNum)

		switch splits[0] {
		case "-test.blockprofile",
			"-test.cpuprofile",
			"-test.memprofile",
			"-test.mutexprofile",
			"-test.trace":
			if !filepath.IsAbs(splits[1]) {
				splits[1] = filepath.Join(outputDir, splits[1])
			}

			fallthrough
		case "-test.coverprofile":
			spec.AdditionalConsoles = append(spec.AdditionalConsoles, splits[1])
			consoleID := len(spec.AdditionalConsoles) - 1
			splits[1] = qemu.AdditionalConsolePath(consoleID)
		default:
			continue
		}

		spec.InitArgs[idx] = strings.Join(splits, "=")
	}
}
