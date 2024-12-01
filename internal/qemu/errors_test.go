// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu_test

import (
	"testing"

	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
)

func TestArgumentErrorIs(t *testing.T) {
	//nolint:testifylint
	assert.ErrorIs(t, error(&qemu.ArgumentError{}), &qemu.ArgumentError{})
	assert.NotErrorIs(t, assert.AnError, &qemu.ArgumentError{})
}

func TestCommandErrorIs(t *testing.T) {
	//nolint:testifylint
	assert.ErrorIs(t, error(&qemu.CommandError{}), &qemu.CommandError{})
	assert.NotErrorIs(t, assert.AnError, &qemu.CommandError{})
}
