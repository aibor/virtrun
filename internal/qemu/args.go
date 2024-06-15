// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package qemu

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Argument is a QEMU argument with or without value.
//
// Its name might be marked to be unique in a list of [Arguments].
type Argument struct {
	name          string
	value         string
	nonUniqueName bool
}

// Name returns the name of the [Argument].
func (a *Argument) Name() string {
	return a.name
}

// Value returns the value of the [Argument].
func (a *Argument) Value() string {
	return a.value
}

// UniqueName returns if the name of the [Argument] must be unique in an
// [Arguments] list.
func (a *Argument) UniqueName() bool {
	return !a.nonUniqueName
}

// String returns a string representation of the argument.
func (a *Argument) String() string {
	return fmt.Sprintf("%s: %s", a.name, a.value)
}

// Equal compares the [Argument]s.
//
// If the name is marked unique, only names are
// compared. Otherwise name and value are compared.
func (a *Argument) Equal(b Argument) bool {
	if a.name != b.name {
		return false
	}

	if a.nonUniqueName {
		return a.value == b.value
	}

	return true
}

// WithValue returns a constructor function that takes a single value and
// returns a new [Argument] with the name of the receiver argument and the
// value passed to the constructor function.
func (a Argument) WithValue() func(string) Argument {
	return func(s string) Argument {
		a := a
		a.value = s

		return a
	}
}

// WithMultiValue is like [Argument.WithValue] but takes multiple values.
func (a Argument) WithMultiValue(separator string) func(...string) Argument {
	return func(s ...string) Argument {
		return a.WithValue()(strings.Join(s, separator))
	}
}

// WithIntValue is like [Argument.WithValue] but takes an integer value instead
// of a string.
func (a Argument) WithIntValue() func(int) Argument {
	return func(i int) Argument {
		return a.WithValue()(strconv.Itoa(i))
	}
}

// UniqueArg returns a new [Argument] with the given name that is marked as
// unique and so can be used in [Arguments] only once.
func UniqueArg(name string) Argument {
	return Argument{
		name: name,
	}
}

// RepeatableArg returns a new [Argument] with the given name that is not
// unique and so can be used in [Arguments] multiple times.
func RepeatableArg(name string) Argument {
	return Argument{
		name:          name,
		nonUniqueName: true,
	}
}

var (
	// Path to the Kernel file.
	ArgKernel = UniqueArg("kernel").WithValue()
	// Path to the initramfs cpio archive file.
	ArgInitrd = UniqueArg("initrd").WithValue()
	// Machine type, depends on the target architecture used.
	ArgMachine = UniqueArg("machine").WithValue()
	// CPU type, depends on the machine type used.
	ArgCPU = UniqueArg("cpu").WithValue()
	// Number of guest CPUs.
	ArgSMP = UniqueArg("smp").WithIntValue()
	// Guest memory in MB.
	ArgMemory = UniqueArg("m").WithIntValue()
	// Display mode.
	ArgDisplay = UniqueArg("display").WithValue()
	// Monitor mode.
	ArgMonitor = UniqueArg("monitor").WithValue()
	// Serial console device for ISA transport type.
	ArgSerial = RepeatableArg("serial").WithValue()
	// Arbitrary device according to QEMUs supported devices.
	ArgDevice = RepeatableArg("device").WithMultiValue(",")
	// Arbitrary cahracter device according to QEMUs supported devices.
	ArgChardev = RepeatableArg("chardev").WithMultiValue(",")
	// Kernel cmdline that is passed to the kernel. Values passed after an "--"
	// are passed to the "init" program as arguments.
	ArgAppend = RepeatableArg("append").WithMultiValue(" ")
)

// Arguments is a list of [Argument]s.
//
// Once all [Argument]s are added, call [Arguments.Build] to compile the
// complete QEMU arguments strings slice.
type Arguments []Argument

// Add adds the given [Argument]s to the list.
func (a *Arguments) Add(e ...Argument) {
	*a = append(*a, e...)
}

// Build compiles the [Argument]s to into a slice of strings which can be used
// with [exec.Command].
//
// It returns an error if any name uniqueness constraints of any [Argument] is
// violated.
func (a Arguments) Build() ([]string, error) {
	s := make([]string, 0, len(a))

	for idx, e := range a {
		if slices.ContainsFunc(a[idx+1:], e.Equal) {
			return nil, fmt.Errorf("colliding args: %s", e.name)
		}

		s = append(s, "-"+e.name)

		if e.value != "" {
			s = append(s, e.value)
		}
	}

	return s, nil
}
