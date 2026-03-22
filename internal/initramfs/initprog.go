// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"

	"github.com/aibor/virtrun/internal/sys"
)

//go:embed init/bin/amd64.gz
var amd64Compressed []byte

//go:embed init/bin/arm64.gz
var arm64Compressed []byte

//go:embed init/bin/riscv64.gz
var riscv64Compressed []byte

// InitProgFor returns the pre-built init executable for the given architecture.
//
// The init binary is supposed to set up the system and execute the file
// "/main".
func InitProgFor(arch sys.Arch) ([]byte, error) {
	var compressed []byte
	switch arch {
	case sys.AMD64:
		compressed = amd64Compressed
	case sys.ARM64:
		compressed = arm64Compressed
	case sys.RISCV64:
		compressed = riscv64Compressed
	default:
		return nil, sys.ErrArchNotSupported
	}

	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("new decompressor: %w", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("decompress: %w", err)
	}

	err = reader.Close()
	if err != nil {
		return nil, fmt.Errorf("close decompressor: %w", err)
	}

	return data, nil
}
