package internal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cavaliergopher/cpio"
)

type Initrd struct {
	*cpio.Writer
}

func (a *Initrd) writeHeader(hdr *cpio.Header) error {
	if err := a.WriteHeader(hdr); err != nil {
		return fmt.Errorf("write header for %s: %v", hdr.Name, err)
	}
	return nil
}

// InitLibStructure adds a directory and various symlinks to it.
//
// The directory is intended to have all dynamic libraries.
func (i *Initrd) InitLibStructure() error {
	dirs := []string{"usr/", "usr/lib/"}
	for _, dir := range dirs {
		err := i.writeHeader(&cpio.Header{
			Name:  dir,
			Mode:  cpio.TypeDir | cpio.ModePerm,
			Links: 2,
		})
		if err != nil {
			return err
		}
	}

	links := []cpio.Header{
		{Name: "lib", Linkname: "usr/lib"},
		{Name: "lib64", Linkname: "usr/lib"},
		{Name: "usr/lib64", Linkname: "lib"},
	}
	for _, link := range links {
		link.Mode = cpio.TypeSymlink | cpio.ModePerm
		link.Size = int64(len(link.Linkname))
		if err := i.writeHeader(&link); err != nil {
			return err
		}
		// Body of a link is the path of the target file.
		if _, err := i.Write([]byte(link.Linkname)); err != nil {
			return fmt.Errorf("write link for %s: %v", link.Name, err)
		}
	}

	return nil
}

// AddRegularFile adds a single regular file to the archive.
//
// If altName is not empty, it is used as file name in the archive.
func (i *Initrd) AddRegularFile(fileName string, altName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("open file: %v", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("read info: %v", err)
	}

	cpioHdr, err := cpio.FileInfoHeader(info, "")
	if err != nil {
		return fmt.Errorf("create header: %v", err)
	}

	if altName != "" {
		cpioHdr.Name = altName
	} else if fileName != cpioHdr.Name {
		cpioHdr.Name = fileName
	}

	if err := i.writeHeader(cpioHdr); err != nil {
		return err
	}

	if _, err := io.Copy(i, file); err != nil {
		return fmt.Errorf("write body: %v", err)
	}

	return nil
}

// CreateInitrd creates a new initrd cpio archive.
//
// The file at initFilePath is added as "/init" to the archive and will be
// executed by the kernel. All additional files are expected to be dynamic
// libraries and put int "/usr/lib". Usual symlinks "/lib", "/lib64" and
// "/usr/lib64" are created to that directory.
//
// The initrd is created in a temporary directory and will be removed once the
// process exits. The function returns the absolute path to the initrd file.
func CreateInitrd(initFilePath string, libs ...string) (string, error) {
	initrdFile, err := os.CreateTemp(filepath.Dir(initFilePath), "go_pidonetest_initrd")
	if err != nil {
		return "", fmt.Errorf("create file")
	}
	defer initrdFile.Close()

	initrd := &Initrd{cpio.NewWriter(initrdFile)}
	defer initrd.Close()

	cleanup := func() {
		initrd.Close()
		os.Remove(initrdFile.Name())
	}

	if err := initrd.AddRegularFile(initFilePath, "init"); err != nil {
		cleanup()
		return "", err
	}

	if len(libs) > 1 {
		// Put all libs in /usr/lib. All other usual lib dirs are symlinking there.
		if err := initrd.InitLibStructure(); err != nil {
			cleanup()
			return "", err
		}
		for _, file := range libs {
			newPath := filepath.Join("/usr/lib/", filepath.Base(file))
			err := initrd.AddRegularFile(file, newPath)
			if err != nil {
				cleanup()
				return "", fmt.Errorf("add lib file %s: %v", newPath, err)
			}
		}
	}

	return initrdFile.Name(), nil
}
