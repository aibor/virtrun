// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit_test

import (
	"bytes"
	"log"
	"testing"

	"github.com/aibor/virtrun/internal/exitcode"
	"github.com/aibor/virtrun/sysinit"
	"github.com/stretchr/testify/assert"
)

func TestExitCodePrinter(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedOut string
		expectedLog string
	}{
		{
			name:        "no error",
			expectedOut: exitcode.Identifier + ": 0\n",
		},
		{
			name:        "an error",
			err:         assert.AnError,
			expectedOut: exitcode.Identifier + ": -1\n",
			expectedLog: "ERROR " + assert.AnError.Error() + "\n",
		},
		{
			name:        "exit error",
			err:         sysinit.ExitError(42),
			expectedOut: exitcode.Identifier + ": 42\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actualOut, actualLog bytes.Buffer

			log.SetFlags(0)
			log.SetOutput(&actualLog)

			sysinit.ExitCodePrinter(&actualOut)(tt.err)

			assert.Equal(t, tt.expectedOut, actualOut.String(), "stdout")
			assert.Equal(t, tt.expectedLog, actualLog.String(), "log")
		})
	}
}
