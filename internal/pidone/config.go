// SPDX-FileCopyrightText: 2026 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package pidone

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/aibor/virtrun/sysinit"
)

// Config provides configuration parameters for the init program.
type Config sysinit.Config

// Encode encodes the config into the format stored in the config file.
func (cfg Config) Encode() ([]byte, error) {
	var buf bytes.Buffer

	err := gob.NewEncoder(&buf).Encode(cfg)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	return buf.Bytes(), nil
}
