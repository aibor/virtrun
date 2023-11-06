package main

import (
	"flag"
	"fmt"
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

	// Catch coverage related paths and adjust them. This is only done until
	// the terminator string "--" is found.
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
		case "-test.gocoverdir":
			splits[1] = "/tmp"
			posArg = strings.Join(splits, "=")
			// TODO: Add handling for all profile file and outputdir flag
		}

		if strings.HasPrefix(posArg, "-") {
			qemuCmd.InitArgs = append(qemuCmd.InitArgs, posArg)
		} else {
			*binaries = append(*binaries, posArg)
		}
	}

	if len(*binaries) < 1 {
		fmt.Fprintln(fs.Output(), "no binary given")
		fs.Usage()
		return fmt.Errorf("no binary given")
	}

	return nil
}
