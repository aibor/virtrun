// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// LimitedUintValue is a uint64 with lower and upper limits.
type LimitedUintValue struct {
	Value        *uint64
	Lower, Upper uint64
}

func (u *LimitedUintValue) String() string {
	if u.Value == nil {
		return "0"
	}

	return strconv.FormatUint(*u.Value, 10)
}

// Set sets [LimitedUintValue] to the given value, if valid.
func (u *LimitedUintValue) Set(s string) error {
	value, err := strconv.ParseUint(s, 10, 0)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	if u.Lower > 0 && value < u.Lower {
		return fmt.Errorf("%d < %d: %w", value, u.Lower, ErrValueOutOfRange)
	}

	if u.Upper > 0 && value > u.Upper {
		return fmt.Errorf("%d > %d: %w", value, u.Upper, ErrValueOutOfRange)
	}

	*u.Value = value

	return nil
}

// PortPair is a pair of ports. if the ports have the same value it is
// represented as a single uint16. Otherwise it is separated by a colon.
type PortPair [2]uint16

func (p *PortPair) String() string {
	if p[0] == p[1] {
		return strconv.Itoa(int(p[1]))
	}

	return fmt.Sprintf("%d:%d", p[0], p[1])
}

// Set sets [PortPair] to the given value, if valid.
func (p *PortPair) Set(input string) error {
	var errs [2]error

	parts := strings.SplitN(input, ":", len(p))

	switch len(parts) {
	case 1:
		p[1], errs[0] = parseUint16(parts[0])
		p[0] = p[1]
	case len(p):
		p[0], errs[0] = parseUint16(parts[0])
		p[1], errs[1] = parseUint16(parts[1])
	default:
		return fmt.Errorf("%w: too many parts", ErrValueOutOfRange)
	}

	return errors.Join(errs[:]...)
}

func parseUint16(s string) (uint16, error) {
	i, err := strconv.ParseUint(s, 10, 16)
	return uint16(i), err
}
