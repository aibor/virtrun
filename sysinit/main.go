// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"errors"
)

// ErrNotPidOne may be returned if the process is expected to be run as PID 1
// but is not.
var ErrNotPidOne = errors.New("process does not have ID 1")

// IsPidOne returns true if the running process has PID 1.
func IsPidOne() bool {
	return getpid() == 1
}

// IsPidOneChild returns true if the running process is a child of the process
// with PID 1.
func IsPidOneChild() bool {
	return getppid() == 1
}

// Poweroff shuts down the system.
//
// Call when done, or deferred right at the beginning of your `TestMain`
// function.
func Poweroff() {
	// Silence the kernel so it does not show up in our test output.
	_ = sysctl("kernel/printk", "0")

	// Use restart instead of poweroff for shutting down the system since it
	// does not require ACPI. The guest system should be started with noreboot.
	if err := reboot(); err != nil {
		PrintError(err)
	}
}

// EnvVars is a map of environment variable values by name.
type EnvVars map[string]string

// Config defines basic system configuration.
type Config struct {
	// MountPoints defines the special file systems that are mounted on init.
	// [MountPoint]s that have the MayFail flag set just produce a warning
	// instead of failing the process.
	MountPoints MountPoints

	// Symlinks is a set of symbolic links that are created on init.
	Symlinks Symlinks

	// Env is a set of environment variables that are added to the process's
	// environment.
	Env EnvVars

	// ConfigureLoopback determines if the loopback interface is brought up on
	// init.
	ConfigureLoopback bool

	// ModulesDir defines the directory that contains kernel modules. They are
	// load on init automatically.
	ModulesDir string
}

// DefaultConfig creates a new default config.
func DefaultConfig() Config {
	return Config{
		// All special file systems required for usual operations, like
		// accessing kernel variables, modifying kernel knobs or accessing
		// devices.
		MountPoints: MountPoints{
			"/dev":                     {FSType: FSTypeDevTmp},
			"/dev/hugepages":           {FSType: FSTypeHugeTlb, MayFail: true},
			"/dev/mqueue":              {FSType: FSTypeMqueue, MayFail: true},
			"/dev/pts":                 {FSType: FSTypeDevPts, MayFail: true},
			"/dev/shm":                 {FSType: FSTypeTmp, MayFail: true},
			"/proc":                    {FSType: FSTypeProc},
			"/run":                     {FSType: FSTypeTmp},
			"/sys/fs/bpf":              {FSType: FSTypeBpf, MayFail: true},
			"/sys/fs/cgroup":           {FSType: FSTypeCgroup2, MayFail: true},
			"/sys/fs/fuse/connections": {FSType: FSTypeFuseCtl, MayFail: true},
			"/sys/fs/pstore":           {FSType: FSTypePstore, MayFail: true},
			"/sys":                     {FSType: FSTypeSys},
			"/sys/kernel/config":       {FSType: FSTypeConfig, MayFail: true},
			"/sys/kernel/debug":        {FSType: FSTypeDebug, MayFail: true},
			"/sys/kernel/security":     {FSType: FSTypeSecurity, MayFail: true},
			"/sys/kernel/tracing":      {FSType: FSTypeTracing, MayFail: true},
			"/tmp":                     {FSType: FSTypeTmp},
		},
		Symlinks: Symlinks{
			"/dev/core":   "/proc/kcore",
			"/dev/fd":     "/proc/self/fd/",
			"/dev/rtc":    "rtc0",
			"/dev/stdin":  "/proc/self/fd/0",
			"/dev/stdout": "/proc/self/fd/1",
			"/dev/stderr": "/proc/self/fd/2",
		},
		Env:               EnvVars{},
		ConfigureLoopback: true,
	}
}

// Main is the entry point for an actual init system.
//
// It sets up the system and ensures proper shut down. Preparation steps are:
// - Guarding itself to be actually PID 1.
// - Setup system poweroff (on function termination!).
// - Load additional kernel modules.
// - Mount all known virtual system file systems.
// - Add well known symlinks in /dev.
// - Bring loopback interface up.
// - Set environment variables.
//
// Once this is done, the given function is run. The function must not
// terminate the process itself (by calling [os.Exit] or panicking)! Otherwise
// the proper system termination is missing and the system will panic due to
// the init program terminating unexpectedly.
//
// The proper termination by this function includes communicating its exit code
// via stdout for consumption by the host process. The exit code returned by
// the given function is used, unless it returned with an error. It is ensured
// that in case of any error a noon-zero exit code is sent (-1).
func Main(cfg Config, fn func() (int, error)) {
	exitCode, err := main(cfg, fn)
	if err != nil {
		// Always print the error before printing the exit code, since
		// output processing stops once exit code line is found and we want
		// to make sure the error can be seen by the user.
		PrintError(err)

		// Always return a non zero exit code in case of error.
		if exitCode == 0 {
			exitCode = -1
		}

		// If this is not the init system, exit the process without shutting
		// down the system.
		if errors.Is(err, ErrNotPidOne) {
			exit(exitCode)
		}
	}

	PrintExitCode(exitCode)
	Poweroff()
}

func main(cfg Config, fn func() (int, error)) (int, error) {
	if !IsPidOne() {
		return -2, ErrNotPidOne
	}

	// Setup the system.
	if err := setup(cfg); err != nil {
		return -1, err
	}

	return fn()
}

func setup(cfg Config) error {
	if cfg.ModulesDir != "" {
		if err := LoadModules(cfg.ModulesDir); err != nil {
			return err
		}
	}

	if cfg.ConfigureLoopback {
		if err := ConfigureLoopbackInterface(); err != nil {
			return err
		}
	}

	if err := MountAll(cfg.MountPoints); err != nil {
		return err
	}

	if err := CreateSymlinks(cfg.Symlinks); err != nil {
		return err
	}

	for key, value := range cfg.Env {
		if err := setenv(key, value); err != nil {
			return err
		}
	}

	return nil
}
