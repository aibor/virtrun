// SPDX-FileCopyrightText: 2026 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Package initprog provides a simple init that does basic system setup (mount
// usual virtual file systems, load modules, bring up loopback interface), runs
// the main executable, and communicates its exit code to the host. It is
// pre-compiled for all supported architectures.
package initprog
