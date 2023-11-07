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

// Install virtrun to gobin directory.
func InstallVirtrun() error {
	path := filepath.Join(env["GOBIN"], "virtrun")
	mod, err := target.Path(path)
	if err != nil {
		return err
	}

	if !mod {
		return nil
	}

	pkg := "github.com/aibor/virtrun/cmd/virtrun"
	version := "latest"
	return sh.RunWith(env, "go", "install", fmt.Sprintf("%s@%s", pkg, version))
}

// Run tests using the package itself.
func Selftest(useInstalled, standalone, verbose bool) error {
	cmdAbsPath, err := filepath.Abs("./cmd/virtrun")
	if err != nil {
		return err
	}
	execCmd := []string{
		"go",
		"run",
		cmdAbsPath,
	}
	tags := []string{"selftest"}
	if useInstalled {
		mg.Deps(InstallVirtrun)
		execCmd = []string{filepath.Join("$GOBIN", "virtrun")}
	}
	if standalone {
		execCmd = append(execCmd, "-standalone")
		tags = append(tags, "standalone")
	}
	if verbose {
		execCmd = append(execCmd, "-verbose")
	}

	args := []string{
		"test",
		"-v",
		"-timeout", "2m",
		"-exec", strings.Join(execCmd, " "),
		"-tags", strings.Join(tags, ","),
		"-cover",
		"-coverprofile", "/tmp/cover.out",
		"-coverpkg", "github.com/aibor/virtrun",
	}
	args = append(args, "./selftest")

	fmt.Printf("go args: %s\n", args)
	return sh.RunWithV(env, "go", args...)
}

// Remove volatile files.
func Clean() error {
	return sh.Rm(env["GOBIN"])
}
