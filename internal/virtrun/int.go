// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"errors"
	"fmt"
	"strconv"
)

var ErrValueOutOfRange = errors.New("value is outside of range")

type LimitedUintFlag struct {
	Value    uint64
	min, max uint64
	unit     string
}

func (u LimitedUintFlag) MarshalText() ([]byte, error) {
	return []byte(strconv.FormatUint(u.Value, 10) + u.unit), nil
}

func (u *LimitedUintFlag) UnmarshalText(text []byte) error {
	value, err := strconv.ParseUint(string(text), 10, 0)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	if u.min > 0 && value < u.min {
		return fmt.Errorf("%d < %d: %w", value, u.min, ErrValueOutOfRange)
	}

	if u.max > 0 && value > u.max {
		return fmt.Errorf("%d > %d: %w", value, u.max, ErrValueOutOfRange)
	}

	u.Value = value

	return nil
}
