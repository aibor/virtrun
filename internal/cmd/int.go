// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"
	"strconv"
)

var ErrValueOutOfRange = errors.New("value is outside of range")

type limitedUintValue struct {
	Value    *uint64
	min, max uint64
}

func (u *limitedUintValue) String() string {
	if u.Value == nil {
		return "0"
	}

	return strconv.FormatUint(*u.Value, 10)
}

func (u *limitedUintValue) Set(s string) error {
	value, err := strconv.ParseUint(s, 10, 0)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	if u.min > 0 && value < u.min {
		return fmt.Errorf("%d < %d: %w", value, u.min, ErrValueOutOfRange)
	}

	if u.max > 0 && value > u.max {
		return fmt.Errorf("%d > %d: %w", value, u.max, ErrValueOutOfRange)
	}

	*u.Value = value

	return nil
}
