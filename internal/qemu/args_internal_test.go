// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package qemu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArgsName(t *testing.T) {
	a := Argument{name: "some"}
	assert.Equal(t, "some", a.Name())
}

func TestArgsValue(t *testing.T) {
	a := Argument{value: "some"}
	assert.Equal(t, "some", a.Value())
}

func TestArgsUniqueName(t *testing.T) {
	a := Argument{nonUniqueName: false}
	b := Argument{nonUniqueName: true}

	assert.True(t, a.UniqueName())
	assert.False(t, b.UniqueName())
}

func TestArgsEqual(t *testing.T) {
	tests := []struct {
		name        string
		a           Argument
		b           Argument
		assertEqual assert.BoolAssertionFunc
	}{
		{
			name:        "both empty",
			a:           Argument{},
			b:           Argument{},
			assertEqual: assert.True,
		},
		{
			name:        "one empty",
			a:           Argument{name: "t"},
			b:           Argument{},
			assertEqual: assert.False,
		},
		{
			name:        "same name",
			a:           Argument{name: "t", value: "5"},
			b:           Argument{name: "t", value: "6"},
			assertEqual: assert.True,
		},
		{
			name:        "same non-unique name",
			a:           Argument{name: "t", value: "5", nonUniqueName: true},
			b:           Argument{name: "t", value: "6", nonUniqueName: true},
			assertEqual: assert.False,
		},
		{
			name:        "same non-unique name and value",
			a:           Argument{name: "t", value: "5", nonUniqueName: true},
			b:           Argument{name: "t", value: "5", nonUniqueName: true},
			assertEqual: assert.True,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assertEqual(t, tt.a.Equal(tt.b), "a")
			tt.assertEqual(t, tt.b.Equal(tt.a), "b")
		})
	}
}

func TestUniqueArg(t *testing.T) {
	tests := []struct {
		name          string
		value         []string
		expectedValue string
	}{
		{
			name:          "empty",
			value:         nil,
			expectedValue: "",
		},
		{
			name:          "single",
			value:         []string{"value"},
			expectedValue: "value",
		},
		{
			name:          "multi",
			value:         []string{"value", "more", "really"},
			expectedValue: "value,more,really",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := Argument{
				name:          "name",
				value:         tt.expectedValue,
				nonUniqueName: false,
			}

			actual := UniqueArg("name", tt.value...)

			assert.Equal(t, expected, actual)
		})
	}
}

func TestRepeatableArg(t *testing.T) {
	tests := []struct {
		name          string
		value         []string
		expectedValue string
	}{
		{
			name:          "empty",
			value:         nil,
			expectedValue: "",
		},
		{
			name:          "single",
			value:         []string{"value"},
			expectedValue: "value",
		},
		{
			name:          "multi",
			value:         []string{"value", "more", "really"},
			expectedValue: "value,more,really",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := Argument{
				name:          "name",
				value:         "value",
				nonUniqueName: true,
			}

			actual := RepeatableArg("name", "value")

			assert.Equal(t, expected, actual)
		})
	}
}
