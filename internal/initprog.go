// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package internal

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
)

// Pre-compile init programs for all supported architectures. Statically linked
// so they can be used on any host platform.
//
//go:generate env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags inits -buildvcs=false -trimpath -ldflags "-s -w" -o init/amd64 ./init/
//go:generate env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags inits -buildvcs=false -trimpath -ldflags "-s -w" -o init/arm64 ./init/

// Embed pre-compiled init programs explicitly to trigger build time errors.
//
//go:embed init/amd64 init/arm64
var _inits embed.FS

// initProgFor returns the pre-built init binary for the arch. The init binary
// is supposed to set up the system and execute the file "/main".
func initProgFor(arch string) (fs.File, error) {
	switch arch {
	case "amd64", "arm64":
		return _inits.Open(filepath.Join("init", arch))
	default:
		return nil, fmt.Errorf("arch not supported: %s", arch)
	}
}
