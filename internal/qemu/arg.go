// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"fmt"
	"slices"
	"strings"
)

// Argument is a QEMU argument with or without value.
//
// Its name might be marked to be unique in a list of [CommandSpec].
type Argument struct {
	name          string
	value         string
	nonUniqueName bool
}

// String implements [fmt.Stringer].
func (a Argument) String() string {
	s := "-" + a.name
	if a.value != "" {
		s += " " + a.value
	}

	return s
}

// Name returns the name of the [Argument].
func (a Argument) Name() string {
	return a.name
}

// Value returns the value of the [Argument].
func (a Argument) Value() string {
	return a.value
}

// UniqueName returns if the name of the [Argument] must be unique in an
// [CommandSpec] list.
func (a Argument) UniqueName() bool {
	return !a.nonUniqueName
}

// Equal compares the [Argument]s.
//
// If the name is marked unique, only names are
// compared. Otherwise name and value are compared.
func (a Argument) Equal(other Argument) bool {
	if a.name != other.name {
		return false
	}

	if a.nonUniqueName {
		return a.value == other.value
	}

	return true
}

// UniqueArg returns a new [Argument] with the given name that is marked as
// unique and so can be used in [CommandSpec] only once.
func UniqueArg(name string, value ...string) Argument {
	return Argument{
		name:  name,
		value: strings.Join(value, ","),
	}
}

// RepeatableArg returns a new [Argument] with the given name that is not
// unique and so can be used in [CommandSpec] multiple times.
func RepeatableArg(name string, value ...string) Argument {
	return Argument{
		name:          name,
		value:         strings.Join(value, ","),
		nonUniqueName: true,
	}
}

// BuildArgumentStrings compiles the [Argument]s to into a slice of strings
// which can be used with [exec.Command].
//
// It returns an error if any name uniqueness constraints of any [Argument] is
// violated.
func BuildArgumentStrings(args []Argument) ([]string, error) {
	argString := make([]string, 0, len(args))

	for idx, arg := range args {
		if i := slices.IndexFunc(args[:idx], arg.Equal); i != -1 {
			return nil, fmt.Errorf(
				"%w: %s, %s",
				ErrArgumentCollision,
				arg.String(),
				args[i].String(),
			)
		}

		argString = append(argString, "-"+arg.name)

		if arg.value != "" {
			argString = append(argString, arg.value)
		}
	}

	return argString, nil
}
