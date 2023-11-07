package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aibor/virtrun/internal/qemu"
)

func parseArgs(args []string, binaries *[]string, qemuCmd *qemu.Command, standalone *bool) error {
	fsName := fmt.Sprintf("%s [flags...] [testbinaries...] [testflags...]", args[0])
	fs := flag.NewFlagSet(fsName, flag.ContinueOnError)

	qemu.AddCommandFlags(fs, qemuCmd)

	fs.BoolVar(
		standalone,
		"standalone",
		*standalone,
		"run test binary as init itself. Use this if the tests has virtrun support built in.",
	)

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	// Catch coverage and profile related paths and adjust them. This is only
	// done until the terminator string "--" is found.
	// Only coverprofile has a relative path to the test pwd. All other profile
	// files are relative to the actual test running and need to be prefixed
	// with -test.outputdir. So, collect them and process them afterwards.
	needsOutputDirPrefix := make([][]string, 0)
	outputDir := ""
	for idx, posArg := range fs.Args() {
		// Once terminator string is found, everything is considered and
		// argument to the init.
		if posArg == "--" {
			qemuCmd.InitArgs = append(qemuCmd.InitArgs, fs.Args()[idx+1:]...)
			break
		}

		splits := strings.Split(posArg, "=")
		switch splits[0] {
		case "-test.coverprofile":
			splits[1] = "/dev/" + qemuCmd.AddExtraFile(splits[1])
			posArg = strings.Join(splits, "=")
		case "-test.blockprofile",
			"-test.cpuprofile",
			"-test.memprofile",
			"-test.mutexprofile",
			"-test.trace":
			needsOutputDirPrefix = append(needsOutputDirPrefix, splits)
			continue
		case "-test.outputdir":
			outputDir = splits[1]
			fallthrough
		case "-test.gocoverdir":
			splits[1] = "/tmp"
			posArg = strings.Join(splits, "=")
		}

		if strings.HasPrefix(posArg, "-") {
			qemuCmd.InitArgs = append(qemuCmd.InitArgs, posArg)
		} else {
			*binaries = append(*binaries, posArg)
		}
	}

	if outputDir != "" {
		for _, arg := range needsOutputDirPrefix {
			path := filepath.Join(outputDir, arg[1])
			arg[1] = "/dev/" + qemuCmd.AddExtraFile(path)
			qemuCmd.InitArgs = append(qemuCmd.InitArgs, strings.Join(arg, "="))
		}
	}

	if len(*binaries) < 1 {
		fmt.Fprintln(fs.Output(), "no binary given")
		fs.Usage()
		return fmt.Errorf("no binary given")
	}

	return nil
}
