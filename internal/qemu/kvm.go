// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package qemu

import (
	"os"
	"runtime"
)

// KVMAvailableFor checks if KVM support is available for the given
// architecture.
func KVMAvailableFor(arch string) bool {
	if runtime.GOARCH != arch {
		return false
	}

	f, err := os.OpenFile("/dev/kvm", os.O_WRONLY, 0)
	_ = f.Close()

	return err == nil
}
