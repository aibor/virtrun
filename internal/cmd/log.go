// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"io"
	"log"
	"log/slog"
)

type logLevel = slog.Level

func setupLogging(w io.Writer, level logLevel) {
	log.SetOutput(w)
	log.SetFlags(log.Lmicroseconds)
	log.SetPrefix("VIRTRUN: ")

	slog.SetLogLoggerLevel(level)
}
