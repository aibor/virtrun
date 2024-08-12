// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package virtrun

import (
	"fmt"
	"strconv"
)

type LimitedUintFlag struct {
	Value    uint
	min, max uint64
	unit     string
}

func (u LimitedUintFlag) MarshalText() ([]byte, error) {
	return []byte(strconv.Itoa(int(u.Value)) + u.unit), nil
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

	u.Value = uint(value)

	return nil
}
