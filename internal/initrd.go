package internal

import (
	"fmt"
	"os"

	"github.com/aibor/initramfs"
)

// CreateInitramfs creates a new initramfs archive.
//
// The file at initFilePath is added as "/init" to the archive and will be
// executed by the kernel. All additional files are put into the directory
// "/files" in the archive.
//
// The initramfs is created in [os.TempDir].  The function returns the absolute
// path to the initramfs. It is the caller's responsibility to the remove the
// file when it is no longer needed.
func CreateInitramfs(initFilePath string, additionalFiles ...string) (string, error) {
	archive := initramfs.New(initFilePath)
	if err := archive.AddFiles(additionalFiles...); err != nil {
		return "", fmt.Errorf("add files: %v", err)
	}
	if err := archive.ResolveLinkedLibs(""); err != nil {
		return "", fmt.Errorf("resolve: %v", err)
	}

	archiveFile, err := os.CreateTemp("", "initramfs")
	if err != nil {
		return "", fmt.Errorf("create file")
	}
	defer archiveFile.Close()

	if err := archive.WriteCPIO(archiveFile); err != nil {
		os.Remove(archiveFile.Name())
		return "", fmt.Errorf("write: %v", err)
	}

	return archiveFile.Name(), nil
}
