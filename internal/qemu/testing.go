// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import "github.com/stretchr/testify/assert"

// ArgumentValueAssertionFunc returns an [assert.ComparisonAssertionFunc] that
// can be used to assert the value of the Argument with the given name.
func ArgumentValueAssertionFunc(
	name string,
	assertion assert.ComparisonAssertionFunc,
) assert.ComparisonAssertionFunc {
	return func(t assert.TestingT, arg1, arg2 any, arg3 ...any) bool {
		args, ok := arg1.([]Argument)
		if !assert.True(t, ok, "first argument should be []Argument") {
			return false
		}

		for _, arg := range args {
			if name != arg.name {
				continue
			}

			return assertion(t, arg.value, arg2, arg3...)
		}

		return assert.Fail(t, "Argument not found")
	}
}
