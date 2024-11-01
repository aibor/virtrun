// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"io"
	"log/slog"
)

func setupLogging(writer io.Writer, debug bool) {
	level := slog.LevelWarn
	if debug {
		level = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(
		writer,
		&slog.HandlerOptions{
			Level: level,
		},
	)))
}
