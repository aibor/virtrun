package qemu

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// SerialConsoleProcessor is used to process input from a serial console and
// write it into a file.
type SerialConsoleProcessor struct {
	name      string
	writePipe *os.File
	readPipe  io.ReadCloser
	output    io.WriteCloser
}

// NewSerialConsoleProcessor creates a new SerialConsoleProcessor that writes
// into a file with the given path. The file is created or truncated, if it
// exists.
func NewSerialConsoleProcessor(serialFile string) (
	*SerialConsoleProcessor,
	error,
) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	f, err := os.Create(serialFile)
	if err != nil {
		return nil, err
	}
	p := &SerialConsoleProcessor{
		name:      serialFile,
		writePipe: w,
		readPipe:  r,
		output:    f,
	}
	return p, nil
}

// Writer returns the writer end of the [os.Pipe].
func (p *SerialConsoleProcessor) Writer() *os.File {
	return p.writePipe
}

// Close closes all file descriptors.
func (p *SerialConsoleProcessor) Close() {
	_ = p.writePipe.Close()
	_ = p.readPipe.Close()
	_ = p.output.Close()
}

// Run process the input. It blocks and returns once [io.EOF] is received,
// which happens when [SerialProcessor.Writer] is closed.
func (p *SerialConsoleProcessor) Run() error {
	if err := clean(p.readPipe, p.output); err != nil {
		return fmt.Errorf("serial processor run %s: %v", p.name, err)
	}
	return nil
}

func clean(r io.Reader, w io.Writer) error {
	buf := make([]byte, 256)
	for {
		n, err := r.Read(buf)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return err
		}
		// Remove carriage returns from the byte stream.
		_, err = w.Write(bytes.ReplaceAll(buf[0:n], []byte("\r"), nil))
		if err != nil {
			return err
		}
	}
}
