// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package initprog

import (
	"embed"
	"fmt"
	"io/fs"
)

// Pre-compile init programs for all supported architectures. Statically linked
// so they can be used on any host platform.
//
//go:generate env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags inits -buildvcs=false -trimpath -ldflags "-s -w" -o amd64 ./init/
//go:generate env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags inits -buildvcs=false -trimpath -ldflags "-s -w" -o arm64 ./init/

// Embed pre-compiled init programs explicitly to trigger build time errors.
//
//go:embed amd64 arm64
var _inits embed.FS

// For returns the pre-built init binary for the arch. The init binary is
// supposed to set up the system and execute the file "/main".
func For(arch string) (fs.File, error) {
	switch arch {
	case "amd64", "arm64":
		return _inits.Open(arch)
	default:
		return nil, fmt.Errorf("arch not supported: %s", arch)
	}
}
