package integrationtesting

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

const URLFmt = "https://github.com/aibor/ci-kernels/raw/master/linux-%s-%s.tgz"

// Known available kernel versions available from
// https://github.com/aibor/ci-kernels.
var (
	Kernel515amd64 = Kernel{"5.15", "amd64"}
	Kernel515arm64 = Kernel{"5.15", "arm64"}
	Kernel61amd64  = Kernel{"6.1", "amd64"}
	Kernel61arm64  = Kernel{"6.1", "arm64"}
	Kernel66amd64  = Kernel{"6.6", "amd64"}
	Kernel66arm64  = Kernel{"6.6", "arm64"}
)

var TestKernels = []Kernel{
	Kernel515amd64,
	Kernel515arm64,
	Kernel61amd64,
	Kernel61arm64,
	Kernel66amd64,
	Kernel66arm64,
}

type Kernel struct {
	Version string
	Arch    string
}

func (k *Kernel) String() string {
	return fmt.Sprintf("%s-%s", k.Version, k.Arch)
}

func (k *Kernel) FileName() string {
	return fmt.Sprintf("vmlinuz-%s-%s", k.Version, k.Arch)
}

func (k *Kernel) Path(dir string) string {
	return filepath.Join(dir, k.FileName())
}

func (k *Kernel) URL() string {
	return fmt.Sprintf(URLFmt, k.Version, k.Arch)
}

func (k *Kernel) Present(dir string) bool {
	_, err := os.Stat(k.Path(dir))
	return err == nil
}

// FetchKernel downloads a pre-built kernel binary from
// https://github.com/aibor/ci-kernels and writes it into the given writer.
// See the repo for available kernel versions and architectures.
func (k *Kernel) Fetch(ctx context.Context, kernelWriter io.Writer) error {
	body, err := fetchArchive(ctx, k.URL())
	if err != nil {
		return err
	}
	defer body.Close()
	return extractKernel(ctx, body, kernelWriter)
}

func fetchArchive(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch archive: %v", err)
	}
	return resp.Body, nil
}

func extractKernel(ctx context.Context, archive io.Reader, to io.Writer) error {
	tarReader := tar.NewReader(archive)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read archive: %v", err)
		}
		// Only care for the kernel file with the known static name.
		if !strings.HasSuffix(header.Name, "/boot/vmlinuz") {
			continue
		}

		_, err = io.Copy(to, tarReader)
		if err != nil {
			return fmt.Errorf("copy kernel: %v", err)
		}
		return nil
	}
	return fmt.Errorf("kernel file not found in archive")
}

// FetchKernels downloads the given kernels in parallel into the given
// directory.
func FetchKernels(dir string, kernels ...Kernel) error {
	ctx, stop := context.WithTimeout(context.Background(), 30*time.Second)
	defer stop()

	eg, ctx := errgroup.WithContext(ctx)
	for _, kernel := range kernels {
		kernel := kernel
		if kernel.Present(dir) {
			continue
		}
		eg.Go(func() error {
			path := kernel.Path(dir)
			file, err := os.Create(path)
			if err != nil {
				return err
			}
			defer file.Close()
			if err := kernel.Fetch(ctx, file); err != nil {
				_ = os.Remove(path)
				return fmt.Errorf("fetch %s: %v", &kernel, err)
			}
			return nil
		})
	}
	return eg.Wait()
}
