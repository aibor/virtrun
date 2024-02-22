// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package initramfs

import "io/fs"

type MockWriter struct {
	Path        string
	RelatedPath string
	Source      fs.File
	Mode        fs.FileMode
	Err         error
}

func (m *MockWriter) WriteRegular(path string, source fs.File, mode fs.FileMode) error {
	m.Path = path
	m.Source = source
	m.Mode = mode
	return m.Err
}

func (m *MockWriter) WriteDirectory(path string) error {
	m.Path = path
	return m.Err
}

func (m *MockWriter) WriteLink(path, target string) error {
	m.Path = path
	m.RelatedPath = target
	return m.Err
}
