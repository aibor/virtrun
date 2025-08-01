// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package sysinit

import (
	"log"
	"slices"
)

type CleanupFunc func() error

type State struct {
	cleanupFns []CleanupFunc
}

func (s *State) Cleanup(fn func() error) {
	s.cleanupFns = append(s.cleanupFns, fn)
}

func (s *State) doCleanup() {
	slices.Reverse(s.cleanupFns)

	for _, fn := range s.cleanupFns {
		if err := fn(); err != nil {
			log.Print("ERROR close: ", err.Error())
		}
	}
}
