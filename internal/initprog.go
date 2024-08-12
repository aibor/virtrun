// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package internal

import (
	"embed"
	"fmt"
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
var _inits embed.FS

// initProgFor returns the pre-built init binary for the arch. The init binary
// is supposed to set up the system and execute the file "/main".
func initProgFor(arch sys.Arch) (fs.File, error) {
	switch arch {
	case sys.AMD64, sys.ARM64, sys.RISCV64:
		f, err := _inits.Open(filepath.Join("bin", arch.String()))
		if err != nil {
			return nil, fmt.Errorf("open: %w", err)
		}

		return f, nil
	default:
		return nil, fmt.Errorf("arch not supported: %s", arch)
	}
}
