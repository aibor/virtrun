//go:build virtrun

package virtrun_test

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aibor/virtrun"
	"github.com/aibor/virtrun/internal/initramfs"
	"github.com/aibor/virtrun/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fetchKernel(path, version, arch string) error {
	urlFmt := "https://github.com/aibor/ci-kernels/raw/master/linux-%s-%s.tgz"
	resp, err := http.Get(fmt.Sprintf(urlFmt, version, arch))
	if err != nil {
		return fmt.Errorf("fetch archive: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	tarReader := tar.NewReader(resp.Body)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read archive: %v", err)
		}
		if !strings.HasSuffix(header.Name, "/boot/vmlinuz") {
			continue
		}

		kernel, err := os.Create(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(kernel, tarReader)
		if err != nil {
			return fmt.Errorf("copy kernel: %v", err)
		}
		return nil
	}
	return fmt.Errorf("kernel file not found")
}

func mustFetchKernel(t testing.TB, path, version, arch string) {
	t.Helper()

	require.NoError(t, fetchKernel(path, version, arch))
}

func TestVirtrun(t *testing.T) {
	t.Setenv("LD_LIBRARY_PATH", "../internal/files/testdata/lib")

	binary, err := filepath.Abs("../internal/files/testdata/bin/main")
	require.NoError(t, err)

	tests := []struct {
		name   string
		arch   string
		rc     int
		errMsg string
	}{
		{
			name: "works on correct arch",
			arch: "amd64",
			rc:   73,
		},
		{
			name: "fails with exec error on wrong arch",
			arch: "arm64",
			rc:   127,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			qemuCmd, err := qemu.CommandFor(tt.arch)
			require.NoError(t, err)

			tmpDir := t.TempDir()
			qemuCmd.Kernel = filepath.Join(tmpDir, "kernel")
			qemuCmd.Initrd = filepath.Join(tmpDir, "initramfs")
			qemuCmd.Verbose = true

			init, err := virtrun.InitFor(tt.arch)
			require.NoError(t, err)

			archive := initramfs.NewWithEmbedded(init)

			err = archive.AddFiles(binary)
			require.NoError(t, err)

			err = archive.AddRequiredSharedObjects()
			require.NoError(t, err)

			file, err := os.Create(qemuCmd.Initrd)
			require.NoError(t, err)

			err = archive.WriteCPIO(file)
			require.NoError(t, err)

			t.Logf("Fetch kernel %s %s", "6.6", tt.arch)
			mustFetchKernel(t, qemuCmd.Kernel, "6.6", tt.arch)
			t.Logf("Fetched kernel")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			t.Cleanup(cancel)
			rc, err := qemuCmd.Run(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.rc, rc)
		})
	}
}
