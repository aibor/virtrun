// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

const sysFileMode = 0o600

// ErrNotPidOne may be returned if the process is expected to be run as PID 1
// but is not.
var ErrNotPidOne = errors.New("process does not have ID 1")

// IsPidOne returns true if the running process has PID 1.
func IsPidOne() bool {
	return os.Getpid() == 1
}

// IsPidOneChild returns true if the running process is a child of the process
// with PID 1.
func IsPidOneChild() bool {
	return os.Getppid() == 1
}

// Poweroff shuts down the system.
//
// Call when done, or deferred right at the beginning of your `TestMain`
// function.
func Poweroff() {
	// Silence the kernel so it does not show up in our test output.
	_ = os.WriteFile("/proc/sys/kernel/printk", []byte("0"), sysFileMode)

	if err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF); err != nil {
		fmt.Fprintf(os.Stderr, "error calling power off: %v\n", err)
	}
	// We just told the system to shutdown. There's no point in staying around.
	os.Exit(0)
}

// Config defines basic system configuration.
type Config struct {
	MountPoints       MountPoints
	Symlinks          Symlinks
	ConfigureLoopback bool
	ModulesDir        string
}

// DefaultConfig creates a new default config.
func DefaultConfig() Config {
	return Config{
		MountPoints: MountPoints{
			{"/proc", FSTypeProc},
			{"/sys", FSTypeSys},
			{"/sys/fs/bpf", FSTypeBpf},
			{"/dev", FSTypeDevTmp},
			{"/run", FSTypeTmp},
			{"/tmp", FSTypeTmp},
		},
		Symlinks: Symlinks{
			"/dev/fd":     "/proc/self/fd/",
			"/dev/stdin":  "/proc/self/fd/0",
			"/dev/stdout": "/proc/self/fd/1",
			"/dev/stderr": "/proc/self/fd/2",
		},
		ConfigureLoopback: true,
	}
}

// Run is the entry point for an actual init system. It prepares the system
// to be used. Preparing steps are:
// - Guarding itself to be actually PID 1.
// - Setup system poweroff on its exit.
// - Mount all known virtual system file systems.
//
// Once this is done, the given function is run. The function must not call
// [os.Exit] itself since the program would not be able to ensure a correct
// system termination.
//
// After that, an exit code is sent to stdout for consumption by the host
// process. The exit code returned by the function is used, unless it returned
// with an error. If the error is an [exec.ExitError], it is parsed and its
// exit code is used. Otherwise the exit code is 127 in case it was never set
// or 126 in case there was an error.
func Run(cfg Config, fn func() (int, error)) error {
	if !IsPidOne() {
		return ErrNotPidOne
	}

	// From here on we can assume we are a system's init program. Termination
	// will lead to system shutdown, or kernel panic, if we do not shutdown
	// correctly.
	defer Poweroff()

	var (
		// Set fall through exit code to non zero value, so it must be set to
		// zero explicitly by the callers function later.
		exitCode    = 127 // Fall through exit code.
		errExitCode = 126 // Exit code that is used in case of errors.
		err         error
		exitErr     *exec.ExitError
	)

	// Setup the error and exit code printing so it is always printed. In
	// case of setup errors, the failure is communicated properly as well.
	defer func() {
		if err != nil {
			// Always print the error before printing the exit code, since
			// output processing stops once exit code line is found and we want
			// to make sure the error can be seen by the user.
			PrintError(os.Stderr, err)
			// Always return a non zero exit code in case of error.
			exitCode = errExitCode
		}

		PrintExitCode(os.Stdout, exitCode)
	}()

	// Setup the system.
	if cfg.ModulesDir != "" {
		if err = LoadModules(cfg.ModulesDir); err != nil {
			return err
		}
	}

	if cfg.ConfigureLoopback {
		if err = ConfigureLoopbackInterface(); err != nil {
			return err
		}
	}

	if err = MountAll(cfg.MountPoints); err != nil {
		return err
	}

	if err = CreateSymlinks(cfg.Symlinks); err != nil {
		return err
	}

	// Run callers function. The returned exit code is irrelevant if any error
	// is returned, because the deferred error handling will override it.
	exitCode, err = fn()
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
		err = nil
	}

	return err
}
