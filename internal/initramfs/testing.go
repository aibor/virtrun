// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package initramfs

import (
	"io/fs"

	"github.com/stretchr/testify/mock"
)

type MockWriter struct {
	mock.Mock
}

func (m *MockWriter) WriteFile(path string, file fs.File) error {
	args := m.Called(path, file)
	return args.Error(0)
}
