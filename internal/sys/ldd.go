// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sys

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"time"
)

const lddTimeout = 5 * time.Second

// Ldd gathers the required shared objects of the ELF file with the given path.
//
// It invokes the "ldd" executable which is expected to be present on the
// system. It returns an [LDDExecError] in case "ldd" is not available or it
// returned with a non-zero exit code. This might be the case if the binary is
// not dynamically linked.
func Ldd(ctx context.Context, path string) ([]string, error) {
	var lddOutput bytes.Buffer

	err := runLdd(ctx, path, &lddOutput)
	if err != nil {
		return nil, err
	}

	var infos ldInfos

	infos.parseFrom(&lddOutput)
	paths := infos.realPaths()

	return paths, nil
}

func runLdd(ctx context.Context, path string, outW io.Writer) error {
	var stderrBuf bytes.Buffer

	ctx, stop := context.WithTimeout(ctx, lddTimeout)
	defer stop()

	cmd := exec.CommandContext(ctx, "ldd", path)
	cmd.Stdout = outW
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		return &LDDExecError{
			Err:    err,
			Stderr: stderrBuf.String(),
		}
	}

	return nil
}

type ldInfos []ldInfo

// parseFrom takes a ldd output, processes each line and adds an [ldInfo] to
// the list.
func (l *ldInfos) parseFrom(lddOutput io.Reader) {
	scanner := bufio.NewScanner(lddOutput)
	for scanner.Scan() {
		var info ldInfo

		info.parseFrom(scanner.Text())

		*l = append(*l, info)
	}
}

// realPaths returns all shared objects that are a real file in the file system.
// So, everything except vdso.
func (l *ldInfos) realPaths() []string {
	var paths []string

	for _, i := range *l {
		switch {
		case filepath.IsAbs(i.name):
			paths = append(paths, i.name)
		case i.path != "":
			paths = append(paths, i.path)
		}
	}

	return paths
}

type ldInfo struct {
	name  string
	path  string
	start uint
}

// parseLddLibPathFrom returns the resolved path to a shared object if the given
// line has one. Empty string if nothing is found.
func (l *ldInfo) parseFrom(line string) {
	// Format for shared objects that reference an absolute path.
	// From glibc rtld.c: _dl_printf ("\t%s => %s (0x%0*zx)\n",
	_, err := fmt.Sscanf(line, "\t%s => %s (0x%x)", &l.name, &l.path, &l.start)
	if err == nil {
		return
	}
	// Format for shared objects that do not reference anything and might be
	// an absolute path already.
	// From glibc rtld.c: _dl_printf ("\t%s (0x%0*zx)\n"
	_, _ = fmt.Sscanf(line, "\t%s (0x%x)", &l.name, &l.start)
}
