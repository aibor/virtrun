package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/aibor/pidonetest/internal/qemu"
)

func parseArgs(args []string, binaries *[]string, qemuCmd *qemu.Command, standalone *bool) error {
	fsName := fmt.Sprintf("%s [flags...] [testbinaries...] [testflags...]", args[0])
	fs := flag.NewFlagSet(fsName, flag.ContinueOnError)

	qemu.AddCommandFlags(fs, qemuCmd)

	fs.BoolVar(
		standalone,
		"standalone",
		*standalone,
		"run test binary as init itself. Use this if the tests has pidonetest support built in.",
	)

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	// Catch coverage related paths and adjust them.
	for _, posArg := range fs.Args() {
		splits := strings.Split(posArg, "=")
		switch splits[0] {
		case "-test.coverprofile":
			splits[1] = "/dev/" + qemuCmd.AddExtraFile(splits[1])
			posArg = strings.Join(splits, "=")
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

	if len(*binaries) < 1 {
		fmt.Fprintln(fs.Output(), "no binary given")
		fs.Usage()
		return fmt.Errorf("no binary given")
	}

	return nil
}
