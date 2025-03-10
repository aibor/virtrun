// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "testing"

func MustAbsoluteFilePath(tb testing.TB, path string) string {
	tb.Helper()

	abs, err := AbsoluteFilePath(path)
	if err != nil {
		tb.Fatalf("failed to get absolute path %s: %v", path, err)
	}

	return abs
}
