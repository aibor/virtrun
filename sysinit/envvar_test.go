// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit_test

import (
	"os"
	"testing"

	"github.com/aibor/virtrun/sysinit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetEnvVars(t *testing.T) {
	t.Cleanup(func() {
		_ = os.Unsetenv("TESTVAR1")
		_ = os.Unsetenv("TESTVAR2")
	})

	err := sysinit.SetEnv(sysinit.EnvVars{
		"TESTVAR1": "42",
		"TESTVAR2": "269",
	})
	require.NoError(t, err)

	assert.Equal(t, "42", os.Getenv("TESTVAR1"), "testvar1")
	assert.Equal(t, "269", os.Getenv("TESTVAR2"), "testvar2")
}
