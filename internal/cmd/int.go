// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strconv"
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
