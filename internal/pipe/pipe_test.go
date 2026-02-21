// SPDX-FileCopyrightText: 2025 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package pipe_test

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aibor/virtrun/internal/pipe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPath(t *testing.T) {
	assert.Equal(t, "/dev/virtrun42", pipe.Path(42))
}

func TestPipes(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		pipes := new(pipe.Pipes)
		err := pipes.Wait(time.Millisecond)
		require.NoError(t, err)

		assert.Zero(t, pipes.Len(), "length")
	})

	t.Run("timeout", func(t *testing.T) {
		pipeReader, _ := io.Pipe()

		pipes := new(pipe.Pipes)
		pipes.Run(&pipe.Pipe{
			Name:        "test",
			InputReader: pipeReader,
			InputCloser: io.NopCloser(pipeReader),
			Output:      io.Discard,
			CopyFunc:    io.Copy,
		})

		err := pipes.Wait(10 * time.Millisecond)
		require.ErrorIs(t, err, pipe.ErrWaitTimeout)

		assert.Equal(t, 1, pipes.Len(), "length")
	})

	t.Run("no output", func(t *testing.T) {
		pipes := new(pipe.Pipes)
		pipes.Run(&pipe.Pipe{
			Name:        "test",
			InputReader: io.NopCloser(bytes.NewReader(nil)),
			InputCloser: io.NopCloser(nil),
			Output:      io.Discard,
			CopyFunc:    io.Copy,
		})

		err := pipes.Wait(100 * time.Millisecond)
		require.ErrorIs(t, err, pipe.ErrNoOutput)

		assert.Equal(t, 1, pipes.Len(), "length")
	})

	t.Run("no output accepted", func(t *testing.T) {
		pipes := new(pipe.Pipes)
		pipes.Run(&pipe.Pipe{
			Name:        "test",
			InputReader: io.NopCloser(bytes.NewReader(nil)),
			InputCloser: io.NopCloser(nil),
			Output:      io.Discard,
			CopyFunc:    io.Copy,
			MayBeSilent: true,
		})

		err := pipes.Wait(100 * time.Millisecond)
		require.NoError(t, err)

		assert.Equal(t, 1, pipes.Len(), "length")
	})

	t.Run("with data", func(t *testing.T) {
		var output bytes.Buffer

		input := "test\ndata deluxe\n"

		pipes := new(pipe.Pipes)
		pipes.Run(&pipe.Pipe{
			Name:        "test",
			InputReader: io.NopCloser(strings.NewReader(input)),
			InputCloser: io.NopCloser(nil),
			Output:      &output,
			CopyFunc:    io.Copy,
		})

		err := pipes.Wait(100 * time.Millisecond)
		require.NoError(t, err)

		assert.Equal(t, input, output.String())
		assert.Equal(t, 1, pipes.Len(), "length")
	})

	t.Run("multiple with data", func(t *testing.T) {
		var output1, output2 bytes.Buffer

		input1 := "more test\ndata deluxe\n"
		input2 := "marvellous\ndata\n"

		pipes := new(pipe.Pipes)
		pipes.Run(&pipe.Pipe{
			Name:        "test1",
			InputReader: io.NopCloser(strings.NewReader(input1)),
			InputCloser: io.NopCloser(nil),
			Output:      &output1,
			CopyFunc:    io.Copy,
		})
		pipes.Run(&pipe.Pipe{
			Name:        "test2",
			InputReader: io.NopCloser(strings.NewReader(input2)),
			InputCloser: io.NopCloser(nil),
			Output:      &output2,
			CopyFunc:    io.Copy,
		})

		err := pipes.Wait(100 * time.Millisecond)
		require.NoError(t, err)

		assert.Equal(t, input1, output1.String(), "pipe 1 output")
		assert.Equal(t, input2, output2.String(), "pipe 2 output")
		assert.Equal(t, 2, pipes.Len(), "length")
	})

	t.Run("multiple with data and no output", func(t *testing.T) {
		var output bytes.Buffer

		input := "test\ndata\ndeluxe\n"

		pipes := new(pipe.Pipes)
		pipes.Run(&pipe.Pipe{
			Name:        "test1",
			InputReader: io.NopCloser(bytes.NewReader(nil)),
			InputCloser: io.NopCloser(nil),
			Output:      io.Discard,
			CopyFunc:    io.Copy,
		})
		pipes.Run(&pipe.Pipe{
			Name:        "test2",
			InputReader: io.NopCloser(strings.NewReader(input)),
			InputCloser: io.NopCloser(nil),
			Output:      &output,
			CopyFunc:    io.Copy,
		})

		err := pipes.Wait(100 * time.Millisecond)
		require.ErrorIs(t, err, pipe.ErrNoOutput)

		assert.Equal(t, input, output.String())
		assert.Equal(t, 2, pipes.Len(), "length")
	})
}

func TestPipes_ByteWritten(t *testing.T) {
	pipes := new(pipe.Pipes)

	pipes.Run(&pipe.Pipe{
		Name:        "one",
		InputReader: io.NopCloser(strings.NewReader("6bytes")),
		InputCloser: io.NopCloser(nil),
		Output:      io.Discard,
		CopyFunc:    io.Copy,
	})

	pipes.Run(&pipe.Pipe{
		Name:        "two",
		InputReader: io.NopCloser(strings.NewReader("07bytes")),
		InputCloser: io.NopCloser(nil),
		Output:      io.Discard,
		CopyFunc:    io.Copy,
	})

	err := pipes.Wait(time.Second)
	require.NoError(t, err)

	expected := map[string]int64{
		"one": 6,
		"two": 7,
	}

	assert.Equal(t, expected, pipes.BytesWritten())
}
