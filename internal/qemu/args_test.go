package qemu_test

import (
	"testing"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArgsEqual(t *testing.T) {
	tests := []struct {
		name  string
		a     qemu.Argument
		b     qemu.Argument
		equal bool
	}{
		{
			name:  "both empty",
			a:     qemu.Argument{},
			b:     qemu.Argument{},
			equal: true,
		},
		{
			name:  "one empty",
			a:     qemu.Argument{Name: "t"},
			b:     qemu.Argument{},
			equal: false,
		},
		{
			name:  "same name",
			a:     qemu.Argument{Name: "t", Value: "5"},
			b:     qemu.Argument{Name: "t", Value: "6"},
			equal: true,
		},
		{
			name:  "same non-unique name",
			a:     qemu.Argument{Name: "t", Value: "5", NonUniqueName: true},
			b:     qemu.Argument{Name: "t", Value: "6", NonUniqueName: true},
			equal: false,
		},
		{
			name:  "same non-unique name and value",
			a:     qemu.Argument{Name: "t", Value: "5", NonUniqueName: true},
			b:     qemu.Argument{Name: "t", Value: "5", NonUniqueName: true},
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
	a := qemu.Argument{Name: "t"}.WithValue()("val")
	e := qemu.Argument{Name: "t", Value: "val"}
	assert.Equal(t, e, a)
}

func TestArgsWithMultiValue(t *testing.T) {
	a := qemu.Argument{Name: "t"}.WithMultiValue("--")("val1", "val2", "val3")
	e := qemu.Argument{Name: "t", Value: "val1--val2--val3"}
	assert.Equal(t, e, a)
}

func TestArgsWithIntValue(t *testing.T) {
	a := qemu.Argument{Name: "t"}.WithIntValue()(99)
	e := qemu.Argument{Name: "t", Value: "99"}
	assert.Equal(t, e, a)
}

func TestArgsAdd(t *testing.T) {
	a := qemu.Arguments{}
	b := qemu.Argument{Name: "t", Value: "99"}
	a.Add(b)
	assert.Equal(t, qemu.Arguments{b}, a)
}

func TestArgsBuild(t *testing.T) {
	t.Run("builds", func(t *testing.T) {
		a := qemu.Arguments{
			qemu.ArgKernel("vmlinuz"),
			qemu.ArgInitrd("boot"),
			qemu.UniqueArg("yes"),
		}
		e := []string{
			"-kernel", "vmlinuz",
			"-initrd", "boot",
			"-yes",
		}
		b, err := a.Build()
		require.NoError(t, err)
		assert.Equal(t, e, b)
	})
	t.Run("collision", func(t *testing.T) {
		a := qemu.Arguments{
			qemu.ArgKernel("vmlinuz"),
			qemu.ArgKernel("bsd"),
		}
		_, err := a.Build()
		assert.Error(t, err)
	})
}
