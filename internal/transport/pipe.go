// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package transport

import (
	"strconv"
)

const pathPrefix = "/dev/virtrun"

// PipePath creates the absolute host pipe path for the given port.
func PipePath(port int) string {
	return pathPrefix + strconv.Itoa(port)
}
