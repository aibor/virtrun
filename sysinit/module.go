// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	moduleTypeUnknown moduleType = ""
	moduleTypePlain   moduleType = ".ko"
	moduleTypeGZIP    moduleType = ".ko.gz"
	moduleTypeXZ      moduleType = ".ko.xz"
	moduleTypeZSTD    moduleType = ".ko.zst"
)

type moduleType string

func parseModuleType(fileName string) moduleType {
	types := []moduleType{
		moduleTypePlain,
		moduleTypeGZIP,
		moduleTypeXZ,
		moduleTypeZSTD,
	}

	for _, typ := range types {
		if strings.HasSuffix(fileName, string(typ)) {
			return typ
		}
	}

	return moduleTypeUnknown
}

// LoadModules loads all files found for the given glob pattern as kernel
// modules.
//
// See [filepath.Glob] for the pattern format.
func LoadModules(pattern string) error {
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("list module files: %w", err)
	}

	for _, file := range files {
		if info, err := os.Stat(file); err == nil && info.IsDir() {
			continue
		}

		if err := LoadModule(file, ""); err != nil {
			return fmt.Errorf("load module %s: %w", file, err)
		}
	}

	return nil
}

// WithModules returns a setup [Func] that wraps [LoadModules] and can
// be used with [Run].
func WithModules(pattern string) Func {
	return func(_ *State) error {
		return LoadModules(pattern)
	}
}

// LoadModule loads the kernel module located at the given path with the given
// parameters.
//
// The file may be compressed. The caller is responsible to ensure the module
// belongs to the running kernel and all dependencies are satisfied.
func LoadModule(path string, params string) error {
	module, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer module.Close()

	return loadModule(module, params)
}

func loadModule(module *os.File, params string) error {
	typ := parseModuleType(module.Name())

	// Try finit_module(2) first, as it is the more comfortable syscall. If it
	// is not available try again with init_module(2).
	err := finitModule(int(module.Fd()), params, finitFlagsFor(typ))
	if !errors.Is(err, errors.ErrUnsupported) {
		return err
	}

	moduleReader, err := newModuleReader(module, typ)
	if err != nil {
		return fmt.Errorf("module reader: %w", err)
	}

	var data bytes.Buffer

	_, err = data.ReadFrom(moduleReader)
	if err != nil {
		return fmt.Errorf("read module: %w", err)
	}

	return initModule(data.Bytes(), params)
}

func newModuleReader(fileReader io.Reader, typ moduleType) (io.Reader, error) {
	switch typ {
	case moduleTypePlain:
		return fileReader, nil
	case moduleTypeGZIP:
		gzipReader, err := gzip.NewReader(fileReader)
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}

		return gzipReader, nil
	default:
		return nil, fmt.Errorf("extension %s: %w", typ, errors.ErrUnsupported)
	}
}

func finitFlagsFor(typ moduleType) finitFlags {
	var flags finitFlags

	if isSupportedFinitCompressionType(typ) {
		flags |= finitFlagCompressedFile
	}

	return flags
}

// isSupportedFinitCompressionType checks if the given extension is one of the
// known extensions finit_module(2) supports.
func isSupportedFinitCompressionType(typ moduleType) bool {
	supportedTypes := []moduleType{
		moduleTypeGZIP,
		moduleTypeXZ,
		moduleTypeZSTD,
	}

	return slices.Contains(supportedTypes, typ)
}
