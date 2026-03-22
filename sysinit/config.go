// SPDX-FileCopyrightText: 2026 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"encoding/gob"
	"fmt"
	"io"
	"net/netip"
	"os"
)

// Config provides configuration parameters.
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

// WithConfigFile returns a setup [Func] that wraps [Config.DecodeFrom] and can
// be used with [Run].
func WithConfigFile(path string, cfg *Config) Func {
	return func(_ *State) error {
		return cfg.DecodeFrom(path)
	}
}
