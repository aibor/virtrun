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
	ArchAMD64  Arch = "amd64"
	ArchARM64  Arch = "arm64"
	ArchNative Arch = Arch(runtime.GOARCH)
)

func (a Arch) String() string {
	return string(a)
}

func (a Arch) IsNative() bool {
	return ArchNative == a
}

func (a Arch) KVMAvailable() bool {
	return qemu.KVMAvailableFor(a.String())
}

func (a Arch) MarshalText() ([]byte, error) {
	return []byte(a), nil
}

func (a *Arch) UnmarshalText(text []byte) error {
	arch := Arch(string(text))
	if arch != ArchAMD64 && arch != ArchARM64 {
		return ErrArchNotSupported
	}

	*a = arch

	return nil
}
