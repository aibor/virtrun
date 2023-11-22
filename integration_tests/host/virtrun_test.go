//go:build integration

package main

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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/aibor/virtrun"
	"github.com/aibor/virtrun/initramfs"
	"github.com/aibor/virtrun/qemu"
)

func fetchKernel(ctx context.Context, path, version, arch string) error {
	url := fmt.Sprintf(
		"https://github.com/aibor/ci-kernels/raw/master/linux-%s-%s.tgz",
		version,
		arch,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("new request (%s, %s): %v", version, arch, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch archive (%s, %s): %v", version, arch, err)
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

func TestVirtrun(t *testing.T) {
	t.Setenv("LD_LIBRARY_PATH", "../../internal/files/testdata/lib")

	binary, err := filepath.Abs("../../internal/files/testdata/bin/main")
	require.NoError(t, err)

	tmpDir := t.TempDir()
	kernelPath := func(arch string) string {
		return filepath.Join(tmpDir, "kernel-"+arch)
	}

	// Fetch kernel in parallel beforehand. Running the whole tests in parallel
	// messes up the debug output, so run the actual tests serialized while
	// speeding up the whole test by pre-fetching the kernels in parallel.
	ctx, stop := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(stop)

	eg, ctx := errgroup.WithContext(ctx)
	for _, arch := range []string{"amd64", "arm64"} {
		arch := arch
		eg.Go(func() error {
			return fetchKernel(ctx, kernelPath(arch), "6.6", arch)
		})
	}
	require.NoError(t, eg.Wait(), "must fetch kernels")

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
			qemuCmd, err := qemu.CommandFor(tt.arch)
			require.NoError(t, err)

			qemuCmd.Kernel = kernelPath(tt.arch)
			qemuCmd.Initrd = filepath.Join(tmpDir, "initramfs-"+tt.arch)
			qemuCmd.Verbose = true

			init, err := virtrun.InitFor(tt.arch)
			require.NoError(t, err)

			archive := initramfs.New(
				initramfs.InitFileVirtual(init),
				initramfs.WithFilesDir("virtrun"),
			)

			err = archive.AddFiles(binary)
			require.NoError(t, err)

			err = archive.AddRequiredSharedObjects()
			require.NoError(t, err)

			file, err := os.Create(qemuCmd.Initrd)
			require.NoError(t, err)

			err = archive.WriteInto(file)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			t.Cleanup(cancel)
			rc, err := qemuCmd.Run(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.rc, rc)
		})
	}
}
