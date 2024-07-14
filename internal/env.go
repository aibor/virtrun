// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package internal

import (
	"os"
	"strings"
)

// GetArch gets the architecture to use for the command.
func GetArch() (Arch, error) {
	arch := ArchNative

	// Allow user to specify architecture by dedicated env var VIRTRUN_ARCH. It
	// can be empty, to suppress the GOARCH lookup and enforce the fallback to
	// the runtime architecture. If VIRTRUN_ARCH is not present, GOARCH will be
	// used. This is handy in case of cross-architecture go test invocations.
	for _, name := range []string{"VIRTRUN_ARCH", "GOARCH"} {
		if v, exists := os.LookupEnv(name); exists {
			// Keep default native arch in case the var is empty.
			if v != "" {
				err := arch.UnmarshalText([]byte(v))
				if err != nil {
					return "", err
				}
			}

			break
		}
	}

	return arch, nil
}

// PrependEnvArgs prepends virtrun arguments from the environment to the given
// list and returns the result. Because those args are prepended, the given
// args have precedence when parsed with [flag].
func PrependEnvArgs(args []string) []string {
	envArgs := strings.Fields(os.Getenv("VIRTRUN_ARGS"))

	return append(envArgs, args...)
}
