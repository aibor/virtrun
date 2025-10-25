// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sys

import (
	"errors"
	"os"
	"runtime"
)

type Arch string

// Supported guest architectures.
const (
	AMD64   Arch = "amd64"
	ARM64   Arch = "arm64"
	RISCV64 Arch = "riscv64"
)

// Native is the architecture of the host. Using the same architecture for the
// guest allows using KVM, if available. Use [KVMAvailable] to check.
const Native Arch = Arch(runtime.GOARCH)

var ErrArchNotSupported = errors.New("architecture not supported")

func (a *Arch) String() string {
	return string(*a)
}

func (a *Arch) IsNative() bool {
	return Native == *a
}

// KVMAvailable checks if KVM support is available for the given architecture.
func (a *Arch) KVMAvailable() bool {
	if !a.IsNative() {
		return false
	}

	f, err := os.OpenFile("/dev/kvm", os.O_WRONLY, 0)
	_ = f.Close()

	return err == nil
}

func (a *Arch) Set(s string) error {
	switch Arch(s) {
	case AMD64, ARM64, RISCV64:
		*a = Arch(s)
	default:
		return ErrArchNotSupported
	}

	return nil
}
