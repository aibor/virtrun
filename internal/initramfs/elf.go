// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package initramfs

import (
	"bufio"
	"bytes"
	"context"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const lddTimeoutSeconds = 5

var (
	// ErrNoInterpreter is returned if no interpreter is found in an ELF file.
	ErrNoInterpreter = errors.New("no interpreter in ELF file")
	// ErrNotELFFile is returned if the file does not have an ELF magic number.
	ErrNotELFFile = errors.New("is not an ELF file")
)

// Ldd gathers the required shared objects of the ELF file with the given path.
// The path must point to an ELF file. [ErrNotELFFile] is returned if it is not
// an ELF file. [ErrNoInterpreter] is returned if no interpreter path is found
// in the ELF file. This is the case if the binary was statically linked.
//
// The objects are searched for in the usual search paths of the
// file's interpreter. Note that the dynamic linker consumes the environment
// variable LD_LIBRARY_PATH, so it can be used to add additional search paths.
//
// Since this implementation executes the ELF interpreter of the given file,
// it should be run only for trusted binaries!
//
// Until version 2.27 the glibc provided ldd worked like this implementation
// and executed the binaries ELF interpreter directly. Since this has some
// security implications and may lead to unintended execution of arbitrary code
// it was changed with version 2.27. With newer versions, ldd tries a set of
// fixed known ELF interpreters. Since they are specific to the glibc build,
// thus Linux distribution specific, it is not feasible for this
// implementation. Because of this the former procedure is used, so use with
// great care!
func Ldd(path string) ([]string, error) {
	interpreter, err := readInterpreter(path)
	if err != nil {
		return nil, err
	}

	infos, err := ldd(interpreter, path)
	if err != nil {
		return nil, err
	}

	paths := infos.realPaths()
	// Append the interpreter itself to the paths, to make sure it is present
	// in the list. Usually, it is already in there pulled in by libc.
	if !slices.Contains(paths, interpreter) {
		paths = append(paths, interpreter)
	}

	return paths, nil
}

// readInterpreter fetches the ELF interpreter path from the ELF file. If the
// file does not have an ELF magic number, [ErrNoELFFile] is returned. If no
// interpreter path is found, [ErrNoInterpreter] is returned.
func readInterpreter(path string) (string, error) {
	elfFile, err := elf.Open(path)
	if err != nil {
		if strings.Contains(err.Error(), "bad magic number") {
			return "", ErrNotELFFile
		}

		return "", err
	}
	defer elfFile.Close()

	for _, prog := range elfFile.Progs {
		if prog.Type != elf.PT_INTERP {
			continue
		}

		buf := make([]byte, prog.Filesz)
		_, err := prog.Open().Read(buf)

		if err != nil && !errors.Is(err, io.EOF) {
			return "", fmt.Errorf("read interpreter: %v", err)
		}
		// Only terminate if the found path is not empty. If there is no other
		// prog with a valid path, it will result in the final ErrNoInterpreter.
		interpreter := unix.ByteSliceToString(buf)
		if interpreter != "" {
			return interpreter, nil
		}
	}

	return "", ErrNoInterpreter
}

// ldd fetches the list of shared objects for the elfFile and populates its
// ldInfos field.
//
// This is how the glibc provided ldd works. The main difference is, that
// it does not try a list of interpreters, but uses the one found in the file
// itself. so, call [elfFile.readInterpreter] before calling ldd.
//
// It returns ErrNoInterpreter if the elfFile has no interpreter set.
func ldd(interpreter, path string) (ldInfos, error) {
	if interpreter == "" {
		return nil, ErrNoInterpreter
	}

	timeout := lddTimeoutSeconds * time.Second

	ctx, stop := context.WithTimeout(context.Background(), timeout)
	defer stop()

	var stdoutBuf, stderrBuf bytes.Buffer

	cmd := exec.CommandContext(ctx, interpreter, "--list", path)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ldd: %v: %s", err, stderrBuf.String())
	}

	var infos ldInfos

	infos.parseFrom(&stdoutBuf)

	return infos, nil
}

type ldInfo struct {
	name  string
	path  string
	start uint
}

type ldInfos []ldInfo

// parseFrom takes a ldd output, processes each line and adds an [ldInfo] to
// the list.
func (l *ldInfos) parseFrom(buf *bytes.Buffer) {
	scanner := bufio.NewScanner(buf)
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
		case i.path != "":
			paths = append(paths, i.path)
		case filepath.IsAbs(i.name):
			paths = append(paths, i.name)
		}
	}

	return paths
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
