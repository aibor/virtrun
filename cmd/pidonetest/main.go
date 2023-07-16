package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/aibor/go-pidonetest/internal"
)

var debugLog *log.Logger

func init() {
	debugLog = log.New(io.Discard, "DEBUG: ", log.LstdFlags)
}

type config struct {
	qemuCmd        internal.QEMUCommand
	testBinaryPath string
}

func run(cmd *exec.Cmd) (int, error) {
	debugLog.Printf("qemu cmd: %s", cmd.String())

	rc := 1

	signalStream := make(chan os.Signal, 1)
	signal.Notify(
		signalStream,
		syscall.SIGABRT,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)

	cmdOut, err := cmd.StdoutPipe()
	if err != nil {
		return rc, fmt.Errorf("get stdout: %v", err)
	}
	defer cmdOut.Close()

	cmdErr, err := cmd.StderrPipe()
	if err != nil {
		return rc, fmt.Errorf("get stderr: %v", err)
	}
	defer cmdErr.Close()

	if err := cmd.Start(); err != nil {
		return rc, fmt.Errorf("run qemu: %v", err)
	}
	p := cmd.Process
	if p != nil {
		defer func() {
			_ = p.Kill()
		}()
	}

	readGroup, rcStream, err := internal.Consume(&internal.Output{
		OutReader: cmdOut,
		ErrReader: cmdErr,
		OutWriter: os.Stdout,
		ErrWriter: os.Stderr,
	})
	if err != nil {
		return rc, fmt.Errorf("start readers: %v", err)
	}

	done := make(chan bool)
	go func() {
		_ = cmd.Wait()
		readGroup.Wait()
		close(done)
	}()

	select {
	case sig := <-signalStream:
		return rc, fmt.Errorf("signal received: %d, %s", sig, sig)
	case rc = <-rcStream:
		return rc, nil
	case <-done:
		return rc, nil
	}
}

func main() {
	cfg := config{
		qemuCmd: internal.QEMUCommand{
			Binary:  "qemu-system-x86_64",
			Kernel:  "/boot/vmlinuz-linux",
			Machine: "q35",
			CPU:     "host",
			Memory:  128,
			NoKVM:   false,
		},
	}

	if !parseFlags(os.Args, &cfg) {
		// Flag already prints errors, so we just exit.
		os.Exit(1)
	}

	libs, err := internal.ResolveLinkedLibs(cfg.testBinaryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving libs (Try again with CGO_ENABLED=0):\n%v", err)
		os.Exit(1)
	}

	cfg.qemuCmd.Initrd, err = internal.CreateInitrd(cfg.testBinaryPath, libs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating intird (Try again with CGO_ENABLED=0):\n%v", err)
		os.Exit(1)
	}

	rc, err := run(cfg.qemuCmd.Cmd())
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error running QEMU command:\n", err)
	} else if err := cfg.qemuCmd.FixSerialFiles(); err != nil {
		fmt.Fprintln(os.Stderr, "Error fixing serial file:\n", err)
	}

	os.Exit(rc)
}
