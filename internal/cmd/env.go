// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// EnvArgs returns virtrun arguments from the environment.
func EnvArgs() []string {
	return strings.Fields(os.Getenv("VIRTRUN_ARGS"))
}

// LocalConfigArgs returns virtrun arguments from a local config file.
//
// The file's format is one argument per line. Environment variables may be used
// and are expanded with [os.ExpandEnv].
func LocalConfigArgs(fsys fs.FS, file string) ([]string, error) {
	conf, err := fs.ReadFile(fsys, file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("read file: %w", err)
	}

	args := []string{}

	expandedConf := os.ExpandEnv(string(conf))
	for line := range strings.SplitSeq(expandedConf, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			args = append(args, line)
		}
	}

	return args, nil
}
