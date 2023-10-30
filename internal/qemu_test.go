package internal_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aibor/pidonetest/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArgs(t *testing.T) {
	next := func(s *[]string) string {
		e := (*s)[0]
		*s = (*s)[1:]
		return e
	}

	t.Run("yes-kvm", func(t *testing.T) {
		q := internal.QEMUCommand{}

		assert.Contains(t, q.Args(), "-enable-kvm")
	})

	t.Run("no-kvm", func(t *testing.T) {
		q := internal.QEMUCommand{
			NoKVM: true,
		}

		assert.NotContains(t, q.Args(), "-enable-kvm")
	})

	t.Run("yes-verbose", func(t *testing.T) {
		q := internal.QEMUCommand{
			Verbose: true,
		}

		assert.NotContains(t, q.Args()[len(q.Args())-1], "loglevel=0")
	})

	t.Run("no-verbose", func(t *testing.T) {
		q := internal.QEMUCommand{}

		assert.Contains(t, q.Args()[len(q.Args())-1], "loglevel=0")
	})

	t.Run("serial files", func(t *testing.T) {
		q := internal.QEMUCommand{
			SerialFiles: []string{
				"/output/file1",
				"/output/file2",
			},
		}
		args := q.Args()
		expected := []string{
			"stdio,id=virtiocon0",
			"file,id=virtiocon1,path=/output/file1",
			"file,id=virtiocon2,path=/output/file2",
		}

		for len(args) > 1 {
			arg := next(&args)
			if arg != "-chardev" {
				continue
			}
			if assert.Greater(t, len(expected), 0, "expected serial files already consumed") {
				assert.Equal(t, next(&expected), next(&args))
			}
		}

		assert.Len(t, expected, 0, "no expected serial files should be left over")
	})

	t.Run("init args", func(t *testing.T) {
		q := internal.QEMUCommand{
			InitArgs: []string{
				"first",
				"second",
				"third",
			},
		}
		args := q.Args()
		expected := " -- first second third"

		var appendValue string
		for len(args) > 1 {
			arg := next(&args)
			if arg == "-append" {
				appendValue = next(&args)
			}
		}

		require.NotEmpty(t, appendValue, "append value must be found")
		assert.Contains(t, appendValue, expected, "append value should contain init args")
	})
}

func TestOutputConsume(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		found   bool
		rc      int
		input   []string
		output  []string
	}{
		{
			name: "panic",
			rc:   126,
			input: []string{
				"[    0.578502] Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000100",
				"[    0.579013] CPU: 0 PID: 76 Comm: init Not tainted 6.4.3-arch1-1 #1 13c144d261447e0acbf2632534d4009bddc4c3ab",
				"[    0.579512] Hardware name: QEMU Standard PC (Q35 + ICH9, 2009), BIOS Arch Linux 1.16.2-1-1 04/01/2014",
			},
			output: []string{""},
		},
		{
			name:    "panic verbose",
			verbose: true,
			rc:      126,
			input: []string{
				"[    0.578502] Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000100",
				"[    0.579013] CPU: 0 PID: 76 Comm: init Not tainted 6.4.3-arch1-1 #1 13c144d261447e0acbf2632534d4009bddc4c3ab",
				"[    0.579512] Hardware name: QEMU Standard PC (Q35 + ICH9, 2009), BIOS Arch Linux 1.16.2-1-1 04/01/2014",
			},
			output: []string{
				"[    0.578502] Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000100",
				"[    0.579013] CPU: 0 PID: 76 Comm: init Not tainted 6.4.3-arch1-1 #1 13c144d261447e0acbf2632534d4009bddc4c3ab",
				"[    0.579512] Hardware name: QEMU Standard PC (Q35 + ICH9, 2009), BIOS Arch Linux 1.16.2-1-1 04/01/2014",
				"",
			},
		},
		{
			name:  "rc",
			found: true,
			rc:    4,
			input: []string{
				"something out",
				"more out",
				fmt.Sprintf(internal.RCFmt, 4),
			},
			output: []string{
				"something out",
				"more out",
				"",
			},
		},
		{
			name: "no rc",
			input: []string{
				"something out",
				"more out",
			},
			output: []string{
				"something out",
				"more out",
				"",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cmdOut := bytes.NewBuffer([]byte(strings.Join(tt.input, "\n")))
			stdOut := bytes.NewBuffer(make([]byte, 0, 512))

			rcParser := internal.NewRCParser(stdOut, tt.verbose)
			done := rcParser.Start()
			_, err := io.Copy(rcParser, cmdOut)
			require.NoError(t, err)
			require.NoError(t, rcParser.Close())

			select {
			case <-done:
				assert.Equal(t, tt.rc, rcParser.RC)
			case <-time.After(100 * time.Millisecond):
				assert.Fail(t, "rcParser did not return in time")
			}
		})
	}
}
