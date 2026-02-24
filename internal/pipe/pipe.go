// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package pipe

import (
	"io"
	"strconv"
)

// CopyFunc defines a function that reads the data from the given reader into
// the given writer.
//
// It may copy the data as is, like [io.Copy], or mutate or filter it as needed.
type CopyFunc func(dst io.Writer, src io.Reader) (int64, error)

var _ CopyFunc = io.Copy

const pathPrefix = "/dev/virtrun"

// Path creates the absolute host pipe path for the given port.
func Path(port int) string {
	return pathPrefix + strconv.Itoa(port)
}
