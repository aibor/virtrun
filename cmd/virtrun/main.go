package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	// TODO: Replace with stdlib slices with go 1.21.
	"golang.org/x/exp/slices"

	"github.com/aibor/virtrun"
	"github.com/aibor/virtrun/initramfs"
)

// buildInitramfs creates an initramfs CPIO archive file in the temporary
// directory with a unique name.
func buildInitramfs(initFile initramfs.InitFile, files []string) (string, error) {
	archive := initramfs.New(initFile)

	if err := archive.AddFiles("virtrun", files...); err != nil {
		return "", fmt.Errorf("add binaries: %v", err)
	}

	if err := archive.AddRequiredSharedObjects(""); err != nil {
		return "", fmt.Errorf("add libs: %v", err)
	}

	return archive.WriteToTempFile("")
}

func run() (int, error) {
	var err error

	cfg := config{
		arch: os.Getenv("GOARCH"),
	}
	if cfg.arch == "" {
		cfg.arch = runtime.GOARCH
	}

	cfg.cmd, err = virtrun.NewCommand(cfg.arch)
	if err != nil {
		return 1, err
	}

	// Preset kernel from environment.
	cfg.cmd.Kernel = os.Getenv("QEMU_KERNEL")

	// ParseArgs already prints errors, so we just exit.
	if err := cfg.parseArgs(os.Args); err != nil {
		if err == flag.ErrHelp {
			return 0, nil
		}
		return 1, nil
	}

	// Do some simple input validation to catch most obvious issues.
	if err := cfg.cmd.Validate(); err != nil {
		return 1, fmt.Errorf("validate qemu command: %v", err)
	}
	if _, err := exec.LookPath(cfg.cmd.Executable); errors.Is(err, os.ErrNotExist) {
		return 1, fmt.Errorf("qemu binary %s: %v", cfg.cmd.Executable, err)
	}
	if _, err := os.Stat(cfg.cmd.Kernel); errors.Is(err, os.ErrNotExist) {
		return 1, fmt.Errorf("kernel file %s doesn't exist.", cfg.cmd.Kernel)
	}
	for _, file := range cfg.binaries {
		if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
			return 1, fmt.Errorf("file %s doesn't exist.", file)
		}
	}

	// In order to be useful with "go test -exec", rewrite the file based flags
	// so the output can be passed from guest to kernel via consoles.
	if !cfg.noGoTestFlagRewrite {
		virtrun.ProcessGoTestFlags(cfg.cmd)
	}

	// Build initramfs for the run.
	var initFile initramfs.InitFile
	if cfg.standalone {
		// In standalone mode, the first binary (which might be the only one)
		// is supposed to work as an init matching our requirements.
		initFile = initramfs.InitFilePath(cfg.binaries[0])
		cfg.binaries = slices.Delete(cfg.binaries, 0, 1)
	} else {
		// In the default wrapped mode a pre-compiled init is used that just
		// execute anything that it finds in the "/virtrun" directory.
		init, err := virtrun.InitFor(cfg.arch)
		if err != nil {
			return 1, err
		}
		initFile = initramfs.InitFileVirtual(init)
	}
	cfg.cmd.Initramfs, err = buildInitramfs(initFile, cfg.binaries)
	if err != nil {
		return 1, fmt.Errorf("initramfs: %v", err)
	}
	defer func() {
		if cfg.keepInitramfs {
			fmt.Fprintf(os.Stderr, "initramfs kept at: %s", cfg.cmd.Initramfs)
		} else {
			_ = os.Remove(cfg.cmd.Initramfs)
		}
	}()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGABRT,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)
	defer cancel()

	rc, err := cfg.cmd.Run(ctx, os.Stdout, os.Stderr)
	if err != nil {
		return rc, fmt.Errorf("run: %v", err)
	}

	return rc, nil
}

func main() {
	rc, err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	os.Exit(rc)
}
