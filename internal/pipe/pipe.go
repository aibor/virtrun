// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package pipe

import (
	"strconv"
)

const pathPrefix = "/dev/virtrun"

// Path creates the absolute host pipe path for the given port.
func Path(port int) string {
	return pathPrefix + strconv.Itoa(port)
}
