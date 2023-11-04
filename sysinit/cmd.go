package sysinit

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

// Run is the entry point for an actual init system. It mounts all relevant
// system file systems and executes files in parallel and shuts the system down
// when done.
func Run(dir string) error {
	if !IsPidOne() {
		return NotPidOneError
	}

	var err error
	defer Poweroff(&err)

	err = MountAll()
	if err != nil {
		return err
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	paths := make([]string, len(files))
	for idx, f := range files {
		paths[idx] = filepath.Join(dir, f.Name())
	}

	rc := 0
	err = ExecParallel(paths, os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		var eerr *exec.ExitError
		if errors.As(err, &eerr) {
			rc = eerr.ExitCode()
		} else {
			rc = 124
		}
		err = nil
	}
	PrintRC(rc)

	return nil
}
