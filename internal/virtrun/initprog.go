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

// Pre-compile init programs for all supported architectures. Statically linked
// so they can be used on any host platform.
//
//go:generate env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags "-s -w" -o bin/amd64 ./init/
//go:generate env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -buildvcs=false -trimpath -ldflags "-s -w" -o bin/arm64 ./init/
//go:generate env CGO_ENABLED=0 GOOS=linux GOARCH=riscv64 go build -buildvcs=false -trimpath -ldflags "-s -w" -o bin/riscv64 ./init/

// Embed pre-compiled init programs explicitly to trigger build time errors.
//
//go:embed bin/*
var initsFS embed.FS

// initProgFor returns the pre-built init binary for the arch.
//
// The init binary is supposed to set up the system and execute the file
// "/main". The returned file name can be opened with initFS.Open.
func initProgFor(arch sys.Arch) (fs.File, error) {
	name := filepath.Join("bin", arch.String())

	file, err := initsFS.Open(name)
	if err != nil {
		return nil, sys.ErrArchNotSupported
	}

	return file, nil
}
