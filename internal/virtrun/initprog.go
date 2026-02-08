// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"embed"
	"io/fs"
	"path/filepath"

	"github.com/aibor/virtrun/internal/sys"
)

// Embed pre-compiled init programs explicitly to trigger build time errors.
//
//go:embed init/bin/amd64
//go:embed init/bin/arm64
//go:embed init/bin/riscv64
var initsFS embed.FS

// InitProgFor returns the pre-built init binary for the arch.
//
// The init binary is supposed to set up the system and execute the file
// "/main". The returned file name can be opened with initFS.Open.
func InitProgFor(arch sys.Arch) (fs.File, error) {
	name := filepath.Join("init", "bin", arch.String())

	file, err := initsFS.Open(name)
	if err != nil {
		return nil, sys.ErrArchNotSupported
	}

	return file, nil
}
