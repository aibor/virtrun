// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"io"
	"log"
	"log/slog"
)

func setupLogging(w io.Writer, debug bool) {
	log.SetOutput(w)
	log.SetFlags(log.Lmicroseconds)
	log.SetPrefix("VIRTRUN: ")

	level := slog.LevelWarn
	if debug {
		level = slog.LevelDebug
	}

	slog.SetLogLoggerLevel(level)
}
