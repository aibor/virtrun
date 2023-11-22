package qemu

import (
	"path/filepath"
	"strings"
)

// ProcessGoTestFlags processes file related go test flags in
// [Command.InitArgs] and changes them, so the guest system's writes end up in
// the host systems file paths.
//
// It scans [Command.InitArgs] for coverage and profile related paths and
// replaces them with console path. The original paths are added as additional
// file descriptors to the [Command].
//
// It is required that the flags are prefixed with "test" and value is
// separated form the flag by "=". This is the format the "go test" tool
// invokes the test binary with.
func ProcessGoTestFlags(c *Command) {
	// Only coverprofile has a relative path to the test pwd and can be
	// replaced immediately. All other profile files are relative to the actual
	// test running and need to be prefixed with -test.outputdir. So, collect
	// them and process them afterwards when "outputdir" is found.
	needsOutputDirPrefix := make([]int, 0)
	outputDir := ""

	for idx, posArg := range c.InitArgs {
		splits := strings.Split(posArg, "=")
		switch splits[0] {
		case "-test.coverprofile":
			splits[1] = "/dev/" + c.AddExtraFile(splits[1])
			c.InitArgs[idx] = strings.Join(splits, "=")
		case "-test.blockprofile",
			"-test.cpuprofile",
			"-test.memprofile",
			"-test.mutexprofile",
			"-test.trace":
			needsOutputDirPrefix = append(needsOutputDirPrefix, idx)
			continue
		case "-test.outputdir":
			outputDir = splits[1]
			fallthrough
		case "-test.gocoverdir":
			splits[1] = "/tmp"
			c.InitArgs[idx] = strings.Join(splits, "=")
		}
	}

	if outputDir != "" {
		for _, argsIdx := range needsOutputDirPrefix {
			splits := strings.Split(c.InitArgs[argsIdx], "=")
			path := filepath.Join(outputDir, splits[1])
			splits[1] = "/dev/" + c.AddExtraFile(path)
			c.InitArgs[argsIdx] = strings.Join(splits, "=")
		}
	}
}
