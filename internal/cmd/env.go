// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"os"
	"strings"
)

// PrependEnvArgs prepends virtrun arguments from the environment to the given
// list and returns the result. Because those args are prepended, the given
// args have precedence when parsed with [flag].
func PrependEnvArgs(args []string) []string {
	envArgs := strings.Fields(os.Getenv("VIRTRUN_ARGS"))
	return append(envArgs, args...)
}
