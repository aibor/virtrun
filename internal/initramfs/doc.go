// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Package initramfs can beused to build simple initramfs CPIO archives. It is
// intended for short lived guests only. The initramfs archives is supposed to
// be as small as possible with only a couple of binaries and their required
// shared libraries.
package initramfs
