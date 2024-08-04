// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package internal

import (
	"runtime"

	"github.com/aibor/virtrun/internal/qemu"
)

type Arch string

const (
	AMD64   Arch = "amd64"
	ARM64   Arch = "arm64"
	RISCV64 Arch = "riscv64"
	Native  Arch = Arch(runtime.GOARCH)
)

func (a Arch) String() string {
	return string(a)
}

func (a Arch) IsNative() bool {
	return Native == a
}

func (a Arch) KVMAvailable() bool {
	return qemu.KVMAvailableFor(a.String())
}

func (a Arch) MarshalText() ([]byte, error) {
	return []byte(a), nil
}

func (a *Arch) UnmarshalText(text []byte) error {
	switch Arch(text) {
	case AMD64, ARM64, RISCV64:
		*a = Arch(text)
	default:
		return ErrArchNotSupported
	}

	return nil
}
