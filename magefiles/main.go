package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"
)

var env map[string]string

func init() {
	env = make(map[string]string)
	gobin, exists := os.LookupEnv("GOBIN")
	if !exists {
		gobin = "./gobin"
	}
	if gobin != "" {
		p, err := filepath.Abs(gobin)
		if err == nil {
			gobin = p
		}
	}
	env["GOBIN"] = gobin
}

// Install pidonetest to gobin directory.
func InstallPidonetest() error {
	path := filepath.Join(env["GOBIN"], "pidonetest")
	mod, err := target.Path(path)
	if err != nil {
		return err
	}

	if !mod {
		return nil
	}

	pkg := "github.com/aibor/pidonetest/cmd/pidonetest"
	version := "latest"
	return sh.RunWith(env, "go", "install", fmt.Sprintf("%s@%s", pkg, version))
}

// Run tests using the package itself.
func Selftest(useInstalled, standalone, verbose bool) error {
	execCmd := []string{
		"go",
		"run",
		"./cmd/pidonetest",
	}
	if useInstalled {
		mg.Deps(InstallPidonetest)
		execCmd = []string{filepath.Join("$GOBIN", "pidonetest")}
	}
	if standalone {
		execCmd = append(execCmd, "-standalone")
	}
	if verbose {
		execCmd = append(execCmd, "-verbose")
	}

	args := []string{
		"test",
		"-v",
		"-exec", strings.Join(execCmd, " "),
	}
	if standalone {
		args = append(args, "-tags", "pidonetest")
	}
	args = append(args, ".")

	fmt.Printf("go args: %s\n", args)
	return sh.RunWithV(env, "go", args...)
}

// Remove volatile files.
func Clean() error {
	return sh.Rm(env["GOBIN"])
}
