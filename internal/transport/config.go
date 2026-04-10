// SPDX-FileCopyrightText: 2026 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package transport

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"net/netip"
	"os"
)

// Config provides configuration parameters for the init program.
type Config struct {
	// Interfaces is a map of Interfaces to configure. If the prefix is the
	// zero value, the interface is only set up without address configuration.
	Interfaces map[string]netip.Prefix
}

// Decode decodes the configuration from the given reader.
func (cfg *Config) Decode(r io.Reader) error {
	err := gob.NewDecoder(r).Decode(cfg)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	return nil
}

// DecodeFrom decodes the config from the file at the given path.
func (cfg *Config) DecodeFrom(path string) error {
	cfgFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	defer cfgFile.Close()

	return cfg.Decode(cfgFile)
}

// Encode encodes the config into the format stored in the config file.
func (cfg *Config) Encode() ([]byte, error) {
	var buf bytes.Buffer

	err := gob.NewEncoder(&buf).Encode(cfg)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	return buf.Bytes(), nil
}
