//go:build integration

package integrationtesting

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aibor/virtrun"
	"github.com/aibor/virtrun/initramfs"
)

func TestHostVirtrunCmd(t *testing.T) {
	t.Setenv("LD_LIBRARY_PATH", "../internal/files/testdata/lib")

	binary, err := filepath.Abs("../internal/files/testdata/bin/main")
	require.NoError(t, err)

	for _, kernel := range TestKernels {
		kernel := kernel
		t.Run(kernel.String(), func(t *testing.T) {
			cmd, err := virtrun.NewCommand(kernel.Arch)
			require.NoError(t, err)

			cmd.Kernel = kernel.Path(KernelCacheDir)
			cmd.Verbose = Verbose

			init, err := virtrun.InitFor(kernel.Arch)
			require.NoError(t, err)

			irfs := initramfs.New(initramfs.InitFileVirtual(init, binary))

			err = irfs.AddRequiredSharedObjects("")
			require.NoError(t, err)

			cmd.Initramfs, err = irfs.WriteToTempFile(t.TempDir())
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			t.Cleanup(cancel)

			rc, err := cmd.Run(ctx, os.Stdout, os.Stderr)
			require.NoError(t, err)

			expectedRC := 73
			if kernel.Arch != runtime.GOARCH {
				expectedRC = 127
			}
			assert.Equal(t, expectedRC, rc)
		})
	}
}
