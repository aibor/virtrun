// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initprog

import (
	"embed"
	"io/fs"
	"path/filepath"

	"github.com/aibor/virtrun/internal/sys"
)

// Embed pre-compiled init programs explicitly to trigger build time errors.
//
//go:embed bin/amd64
//go:embed bin/arm64
//go:embed bin/riscv64
var initsFS embed.FS

// For returns the pre-built init binary for the arch.
//
// The init binary is supposed to set up the system and execute the file
// "/main".
func For(arch sys.Arch) (fs.File, error) {
	name := filepath.Join("bin", arch.String())

	file, err := initsFS.Open(name)
	if err != nil {
		return nil, sys.ErrArchNotSupported
	}

	return file, nil
}
