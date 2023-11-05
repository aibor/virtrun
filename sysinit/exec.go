package sysinit

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Exec executes the given file wit the given arguments. Output and errors are
// written to the given writers immediately. Might return [exec.ExitError].
func Exec(path string, args []string, outWriter, errWriter io.Writer) error {
	cmd := exec.Command(path, args...)
	cmd.Stdout = outWriter
	cmd.Stderr = errWriter
	return cmd.Run()
}

// ExecParallel executes the given files in parallel. Each is called with the
// given args. Output of the commands is written to the given out and err
// writers once the command exited. If there is only a single path given,
// output is printed unbuffered. It respects [runtime.GOMAXPROCS] and does run
// max the number set in parallel. Might return [exec.ExitError].
func ExecParallel(paths []string, args []string, outW, errW io.Writer) error {
	// Fastpath.
	switch len(paths) {
	case 0:
		return nil
	case 1:
		return Exec(paths[0], args, os.Stdout, os.Stderr)
	}

	var (
		writers   sync.WaitGroup
		outStream = make(chan []byte)
		errStream = make(chan []byte)
		addWriter = func(writer io.Writer, byteStream <-chan []byte) {
			writers.Add(1)
			go func(w io.Writer, r <-chan []byte) {
				defer writers.Done()
				for b := range r {
					fmt.Fprint(w, string(b))
				}
			}(writer, byteStream)
		}
	)

	addWriter(outW, outStream)
	addWriter(errW, errStream)

	eg := errgroup.Group{}
	eg.SetLimit(runtime.GOMAXPROCS(0))
	for _, path := range paths {
		path := path
		eg.Go(func() error {
			var outBuf, errBuf bytes.Buffer
			err := Exec(path, args, &outBuf, &errBuf)
			outStream <- outBuf.Bytes()
			errStream <- errBuf.Bytes()
			return err
		})
	}

	err := eg.Wait()
	close(outStream)
	close(errStream)
	writers.Wait()

	return err
}
