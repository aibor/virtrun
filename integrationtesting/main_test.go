//go:build integration

package integrationtesting

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

var (
	KernelCacheDir = "./kernels"
	Verbose        bool
)

func TestMain(m *testing.M) {
	flag.StringVar(
		&KernelCacheDir,
		"kernelDir",
		KernelCacheDir,
		"directory to store kernels in",
	)
	flag.BoolVar(
		&Verbose,
		"verbose",
		Verbose,
		"show complete guest output",
	)
	flag.Parse()

	os.MkdirAll(KernelCacheDir, 0755)

	// Pre-fetch kernels in parallel, to speed up this process
	if err := FetchKernels(KernelCacheDir, TestKernels...); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to pre-fetch kernels: %v", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}
