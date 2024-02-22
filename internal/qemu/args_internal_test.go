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
		name  string
		a     Argument
		b     Argument
		equal bool
	}{
		{
			name:  "both empty",
			a:     Argument{},
			b:     Argument{},
			equal: true,
		},
		{
			name:  "one empty",
			a:     Argument{name: "t"},
			b:     Argument{},
			equal: false,
		},
		{
			name:  "same name",
			a:     Argument{name: "t", value: "5"},
			b:     Argument{name: "t", value: "6"},
			equal: true,
		},
		{
			name:  "same non-unique name",
			a:     Argument{name: "t", value: "5", nonUniqueName: true},
			b:     Argument{name: "t", value: "6", nonUniqueName: true},
			equal: false,
		},
		{
			name:  "same non-unique name and value",
			a:     Argument{name: "t", value: "5", nonUniqueName: true},
			b:     Argument{name: "t", value: "5", nonUniqueName: true},
			equal: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.equal {
				assert.True(t, tt.a.Equal(tt.b), "a")
				assert.True(t, tt.b.Equal(tt.a), "b")
			} else {
				assert.False(t, tt.a.Equal(tt.b), "a")
				assert.False(t, tt.b.Equal(tt.a), "b")
			}
		})
	}
}

func TestArgsWithValue(t *testing.T) {
	a := Argument{name: "t"}.WithValue()("val")
	e := Argument{name: "t", value: "val"}
	assert.Equal(t, e, a)
}

func TestArgsWithMultiValue(t *testing.T) {
	a := Argument{name: "t"}.WithMultiValue("--")("val1", "val2", "val3")
	e := Argument{name: "t", value: "val1--val2--val3"}
	assert.Equal(t, e, a)
}

func TestArgsWithIntValue(t *testing.T) {
	a := Argument{name: "t"}.WithIntValue()(99)
	e := Argument{name: "t", value: "99"}
	assert.Equal(t, e, a)
}
