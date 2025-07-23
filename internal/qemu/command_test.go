// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu_test

import (
	"testing"

	"github.com/aibor/virtrun/internal/pipe"
	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
)

func TestCommmandAddExtraFile(t *testing.T) {
	s := qemu.CommandSpec{}
	d1 := s.AddConsole("test")
	d2 := s.AddConsole("real")

	assert.Equal(t, pipe.Path(1), d1)
	assert.Equal(t, pipe.Path(2), d2)
	assert.Equal(t, []string{"test", "real"}, s.AdditionalConsoles)
}
