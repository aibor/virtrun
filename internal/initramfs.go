package internal

import (
	"fmt"
	"os"

	"github.com/aibor/initramfs"
)

// Initramfs represents an [Initramfs.Archive] with added function to write
// to a tempfile.
type Initramfs struct {
	*initramfs.Archive
}

// NewInitramfs creates a new initramfs archive.
//
// The file at initFilePath is added as "/init" to the archive and will be
// executed by the kernel.
func NewInitramfs(initFilePath string) *Initramfs {
	return &Initramfs{
		Archive: initramfs.New(initFilePath),
	}
}

// Write resolves ELF dynamically linked libraries of all currently added files
// and writes the initramfs to a file in [os.TempDir]. It is the caller's
// responsibility to remove the file when it is no longer needed.
func (i *Initramfs) Write() (string, error) {
	var err error
	if err := i.ResolveLinkedLibs(""); err != nil {
		return "", fmt.Errorf("resolve: %v", err)
	}

	archiveFile, err := os.CreateTemp("", "initramfs")
	if err != nil {
		return "", fmt.Errorf("create file")
	}
	defer archiveFile.Close()

	if err := i.WriteCPIO(archiveFile); err != nil {
		_ = os.Remove(archiveFile.Name())
		return "", fmt.Errorf("write: %v", err)
	}

	return archiveFile.Name(), nil
}
