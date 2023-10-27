package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aibor/initramfs"
)

// CreateInitramfs creates a new initramfs archive.
//
// The file at initFilePath is added as "/init" to the archive and will be
// executed by the kernel. All additional files are put into the directory
// "/files" in the archive.
//
// The initramfs is created in a temporary directory and will be removed once
// the process exits. The function returns the absolute path to the initramfs
// file.
func CreateInitramfs(initFilePath string, additionalFiles ...string) (string, error) {
	archiveFile, err := os.CreateTemp(
		filepath.Dir(initFilePath),
		"go_pidonetest_initramfs",
	)
	if err != nil {
		return "", fmt.Errorf("create file")
	}
	defer archiveFile.Close()

	archive := initramfs.New(initFilePath)
	if err != nil {
		return "", fmt.Errorf("new archive: %v", err)
	}
	if err := archive.AddFiles(additionalFiles...); err != nil {
		return "", fmt.Errorf("add files: %v", err)
	}
	if err := archive.ResolveLinkedLibs(""); err != nil {
		return "", fmt.Errorf("resolve: %v", err)
	}

	if err := archive.WriteCPIO(archiveFile); err != nil {
		return "", fmt.Errorf("write: %v", err)
	}

	return archiveFile.Name(), nil
}
