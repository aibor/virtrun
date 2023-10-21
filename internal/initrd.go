package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aibor/go-initrd"
)

// CreateInitrd creates a new initrd cpio archive.
//
// The file at initFilePath is added as "/init" to the archive and will be
// executed by the kernel. All additional files are put into the directory
// "/files" in the archive.
//
// The initrd is created in a temporary directory and will be removed once the
// process exits. The function returns the absolute path to the initrd file.
func CreateInitrd(initFilePath string, additionalFiles ...string) (string, error) {
	initrdFile, err := os.CreateTemp(filepath.Dir(initFilePath), "go_pidonetest_initrd")
	if err != nil {
		return "", fmt.Errorf("create file")
	}
	defer initrdFile.Close()

	i := initrd.New(initFilePath, additionalFiles...)
	if err != nil {
		return "", fmt.Errorf("mkinitrd: %v", err)
	}
	if err := i.ResolveLinkedLibs(initrd.NewELFLibResolver()); err != nil {
		return "", fmt.Errorf("resolve: %v", err)
	}

	writer := initrd.NewWriter(initrdFile)
	defer writer.Close()

	if err := i.WriteTo(writer); err != nil {
		return "", fmt.Errorf("write: %v", err)
	}

	return initrdFile.Name(), nil
}
