package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
)


func parseFlags(args []string, qemuCmd *QEMUCommand, testBinaryPath *string) bool {
	fs := flag.NewFlagSet(fmt.Sprintf("%s [flags...] [testbinary] [testflags...]", args[0]), flag.ContinueOnError)

	fs.StringVar(
		&qemuCmd.Binary,
		"qemu-bin",
		"qemu-system-x86_64",
		"QEMU binary to use",
	)

	fs.StringVar(
		&qemuCmd.Kernel,
		"kernel",
		"/boot/vmlinuz-linux",
		"path to kernel to use",
	)

	fs.StringVar(
		&qemuCmd.Machine,
		"machine",
		"q35",
		"QEMU machine type to use",
	)

	fs.BoolVar(
		&qemuCmd.NoKVM,
		"nokvm",
		false,
		"disable hardware support",
	)

	fs.Func(
		"memory",
		"memory (in MB) for the QEMU VM (default 128MB)",
		func(s string) error {
			mem, err := strconv.ParseUint(s, 10, 16)
			if err != nil {
				return err
			}
			if mem < 128 {
				return fmt.Errorf("less than 128 MB is not sufficient")
			}

			qemuCmd.Memory = uint16(mem)

			return nil
		},
	)

	if err := fs.Parse(args[1:]); err != nil {
		return false
	}

	posArgs := fs.Args()
	if len(posArgs) < 1 {
		fmt.Fprintln(fs.Output(), "no testbinary given")
		fs.Usage()
		return false
	}

	*testBinaryPath = posArgs[0]

	if len(posArgs) > 1 {
		qemuCmd.TestArgs = append(qemuCmd.TestArgs, posArgs[1:]...)
	}

	return true
}

func run(qemuCmd *QEMUCommand, testBinaryPath string) (int, error) {
	libs, err := resolveLinkedLibs(testBinaryPath)
	if err != nil {
		return 1, err
	}
	if len(libs) > 0 {
		return 1, fmt.Errorf("Test binary must not be linked, but is linked to: % s. Try with CGO_ENABLED=0", libs)
	}

	additional := strings.Split(LibSearchPaths, ":")
	additional = append(additional, libs...)
	initrdFilePath, err := createInitrd(testBinaryPath, additional...)
	if err != nil {
		return 1, fmt.Errorf("create initrd: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(initrdFilePath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to remove initrd file: %s: %v\n", initrdFilePath, err)
		}
	}()

	qemuCmd.Initrd = initrdFilePath

	cmd := qemuCmd.Cmd()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 1, fmt.Errorf("get stdout: %v", err)
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("run qemu: %v", err)
	}
	p := cmd.Process
	if p != nil {
		defer func() {
			_ = p.Kill()
		}()
	}

	done := make(chan bool)
	go func() {
		_ = cmd.Wait()
		close(done)
	}()

	rcStream := make(chan int, 1)
	readGroup := sync.WaitGroup{}
	readGroup.Add(1)
	go func() {
		defer readGroup.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			var rc int
			if _, err := fmt.Sscanf(line, "GO_PIDONETEST_RC: %d", &rc); err != nil {
				fmt.Println(line)
				continue
			}
			if len(rcStream) == 0 {
				rcStream <- rc
			}
		}
	}()

	signalStream := make(chan os.Signal, 1)
	signal.Notify(signalStream, syscall.SIGABRT, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)

	rc := 1

	select {
	case sig := <-signalStream:
		_ = stdout.Close()
		_ = cmd.Process.Kill()
		close(done)
		return rc, fmt.Errorf("signal received: %d, %s", sig, sig)
	case <-done:
		break
	}

	_ = os.Remove(initrdFilePath)
	_ = stdout.Close()
	readGroup.Wait()
	if len(rcStream) == 1 {
		rc = <-rcStream
	}
	return rc, nil
}

func main() {
	var qemuCmd QEMUCommand
	var testBinaryPath string

	if !parseFlags(os.Args, &qemuCmd, &testBinaryPath) {
		// Flag already prints errors, so we just exit.
		os.Exit(1)
	}

	rc, err := run(&qemuCmd, testBinaryPath)
	if err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}

	os.Exit(rc)
}
