// SPDX-FileCopyrightText: 2026 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aibor/virtrun/internal/sys"
)

const minAdditionalFileDescriptor = 3

const (
	machineTypeMicroVM = "microvm"
	machineTypePC      = "pc"
	machineTypeQ35     = "q35"
	machineTypeVirt    = "virt"
)

// CommandSpec defines the parameters for a [Command].
type CommandSpec struct {
	// Path to the qemu-system binary
	Executable string

	// Path to the kernel to boot. The kernel should have Virtio-MMIO support
	// compiled in. If not, set the NoVirtioMMIO flag.
	Kernel string

	// Path to the initramfs to boot with. This is supposed to be a Initramfs
	// built with the initramfs sub package with an init that is built with
	// the sysinit sub package.
	Initramfs string

	// QEMU machine type to use. Depends on the QEMU binary used.
	Machine string

	// CPU type to use. Depends on machine type and QEMU binary used.
	CPU string

	// Number of CPUs for the guest.
	SMP uint64

	// Memory for the machine in MB.
	Memory uint64

	// Disable KVM support.
	NoKVM bool

	// Transport type for IO. This depends on machine type and the kernel.
	// TransportTypeIsa should always work, but will give only one slot for
	// microvm machine type. ARM type virt does not support ISA type at all.
	TransportType TransportType

	// ExtraArgs are  extra arguments that are passed to the QEMU command.
	// They must not interfere with the essential arguments set by the command
	// itself or an error will be returned on [Command.Run].
	ExtraArgs []Argument

	// Additional files attached to consoles besides the default one used for
	// stdout. They will be present in the guest system as "/dev/ttySx" or
	// "/dev/hvcx" where x is the index of the slice + 1.
	AdditionalConsoles []string

	// Arguments to pass to the init binary.
	InitArgs []string

	// Increase guest kernel logging.
	Verbose bool
}

// AddDefaultsFor adds architecture specific default values to the given spec if
// the fields are not set yet.
func (s *CommandSpec) AddDefaultsFor(arch sys.Arch) error {
	var (
		executable    string
		machine       string
		transportType TransportType
	)

	switch arch {
	case sys.AMD64:
		executable = "qemu-system-x86_64"
		machine = machineTypeQ35
		transportType = TransportTypePCI
	case sys.ARM64:
		executable = "qemu-system-aarch64"
		machine = machineTypeVirt
		transportType = TransportTypeMMIO
	case sys.RISCV64:
		executable = "qemu-system-riscv64"
		machine = machineTypeVirt
		transportType = TransportTypeMMIO
	default:
		return sys.ErrArchNotSupported
	}

	if s.Executable == "" {
		s.Executable = executable
	}

	if s.Machine == "" {
		s.Machine = machine
	}

	if s.TransportType == "" {
		s.TransportType = transportType
	}

	if !s.NoKVM {
		s.NoKVM = !arch.KVMAvailable()
	}

	return nil
}

// RewriteGoTestFlagsPath processes file related go test flags so their file
// path are correct for use in the guest system.
//
// It is required that the flags given in the format "-test.<key>=<value>". This
// is the format the "go test" tool invokes the test executable with.
//
// Each file path is replaced with a path to a serial console.
func (s *CommandSpec) RewriteGoTestFlagsPath() {
	const splitNum = 2

	outputDir := ""

	for idx, posArg := range s.InitArgs {
		splits := strings.SplitN(posArg, "=", splitNum)
		switch splits[0] {
		case "-test.outputdir":
			outputDir = splits[1]
			fallthrough
		case "-test.gocoverdir":
			splits[1] = "/tmp"
		default:
			continue
		}

		s.InitArgs[idx] = strings.Join(splits, "=")
	}

	// Only coverprofile has a relative path to the test pwd and can be
	// replaced immediately. All other profile files are relative to the actual
	// test running and need to be prefixed with -test.outputdir. So, collect
	// them and process them afterwards when "outputdir" is found.
	for idx, posArg := range s.InitArgs {
		splits := strings.SplitN(posArg, "=", splitNum)

		switch splits[0] {
		case "-test.blockprofile",
			"-test.cpuprofile",
			"-test.memprofile",
			"-test.mutexprofile",
			"-test.trace":
			if !filepath.IsAbs(splits[1]) {
				splits[1] = filepath.Join(outputDir, splits[1])
			}

			fallthrough
		case "-test.coverprofile":
			s.AdditionalConsoles = append(s.AdditionalConsoles, splits[1])
			consoleID := len(s.AdditionalConsoles) - 1
			splits[1] = AdditionalConsolePath(consoleID)
		default:
			continue
		}

		s.InitArgs[idx] = strings.Join(splits, "=")
	}
}

// Validate checks for known incompatibilities.
func (s *CommandSpec) Validate() error {
	if !s.TransportType.isKnown() {
		return &ArgumentError{
			"unknown transport type: " + s.TransportType.String(),
		}
	}

	switch s.Machine {
	case machineTypeMicroVM:
		switch {
		case s.TransportType == TransportTypePCI:
			return &ArgumentError{"microvm does not support pci transport"}
		case s.TransportType == TransportTypeISA &&
			len(s.AdditionalConsoles) > 0:
			return &ArgumentError{
				"microvm supports only one isa serial port, used for stdio",
			}
		}
	case machineTypeVirt:
		if s.TransportType == TransportTypeISA {
			return &ArgumentError{"virt requires virtio-mmio"}
		}
	case machineTypeQ35, machineTypePC:
		if s.TransportType == TransportTypeMMIO {
			return &ArgumentError{
				s.Machine + " does not work with virtio-mmio",
			}
		}
	}

	return nil
}

// numFDConsoles returns the number of consoles attached via additional file
// descriptors. It is the number of additional consoles given by the user plus
// an additional one for a separate stdout channel.
func (s *CommandSpec) numFDConsoles() int {
	return len(s.AdditionalConsoles) + 1
}

// arguments compiles the argument list for the QEMU command.
func (s *CommandSpec) arguments() []Argument {
	args := []Argument{
		UniqueArg("kernel", s.Kernel),
		UniqueArg("initrd", s.Initramfs),
	}

	if s.Machine != "" {
		args = append(args, UniqueArg("machine", s.Machine))
	}

	if s.CPU != "" {
		args = append(args, UniqueArg("cpu", s.CPU))
	}

	if s.SMP != 0 {
		args = append(args, UniqueArg("smp", strconv.FormatUint(s.SMP, 10)))
	}

	if s.Memory != 0 {
		args = append(args, UniqueArg("m", strconv.FormatUint(s.Memory, 10)))
	}

	if !s.NoKVM {
		args = append(args, UniqueArg("enable-kvm", ""))
	}

	sharedDevices := map[TransportType]string{
		TransportTypePCI:  "virtio-serial-pci,max_ports=8",
		TransportTypeMMIO: "virtio-serial-device,max_ports=8",
	}
	if value, exists := sharedDevices[s.TransportType]; exists {
		args = append(args, RepeatableArg("device", value))
	}

	// Add stdout console.
	args = s.appendConsoleArgs(args, consoleArg{
		id:      "stdio",
		backend: "stdio",
	})

	// Write console output to file descriptors. Those are provided by the
	// [exec.Cmd.ExtraFiles].
	for idx := range s.numFDConsoles() {
		// FDs 0, 1, 2 are standard in, out, err, so start at 3.
		path := fdPath(minAdditionalFileDescriptor + idx)
		args = s.appendConsoleArgs(args, consoleArg{
			id:      fmt.Sprintf("con%d", idx),
			backend: "file",
			opts:    []string{"path=" + path},
		})
	}

	args = append(args,
		// Disable video output.
		UniqueArg("display", "none"),
		// Disable QEMU monitor.
		UniqueArg("monitor", "none"),
		// Guest must not reboot.
		UniqueArg("no-reboot"),
		// Disable all default devices.
		UniqueArg("nodefaults"),
		// Do not load any user config files.
		UniqueArg("no-user-config"),
	)

	args = append(args, s.ExtraArgs...)

	kernelCmdline := strings.Join(s.kernelCmdlineArgs(), " ")
	args = append(args, RepeatableArg("append", kernelCmdline))

	return args
}

// kernelCmdlineArgs reruns the kernel cmdline arguments.
func (s *CommandSpec) kernelCmdlineArgs() []string {
	cmdline := []string{
		"console=" + s.TransportType.ConsoleDeviceName(0),
		"panic=-1",
		"mitigations=off",
		"initcall_blacklist=ahci_pci_driver_init",
	}

	// ACPI is necessary for SMP. With a single CPU, we can disable it to speed
	// up the boot considerably.
	if s.SMP == 1 {
		cmdline = append(cmdline, "acpi=off")
	}

	if s.Verbose {
		cmdline = append(cmdline, "debug")
	} else {
		cmdline = append(cmdline, "quiet")
	}

	if len(s.InitArgs) > 0 {
		cmdline = append(cmdline, "--")
		cmdline = append(cmdline, s.InitArgs...)
	}

	return cmdline
}

type consoleArg struct {
	id      string
	backend string
	opts    []string
}

func (s *CommandSpec) appendConsoleArgs(
	args []Argument,
	console consoleArg,
) []Argument {
	var devArg Argument

	switch s.TransportType {
	case TransportTypeISA:
		devArg = RepeatableArg("serial", "chardev:"+console.id)
	case TransportTypePCI, TransportTypeMMIO:
		devArg = RepeatableArg("device", "virtconsole,chardev="+console.id)
	default: // Ignore invalid transport types.
		return args
	}

	chardevOpts := make([]string, 0, len(console.opts))
	chardevOpts = append(chardevOpts, console.backend, "id="+console.id)
	chardevOpts = append(chardevOpts, console.opts...)

	chardevArg := RepeatableArg("chardev", strings.Join(chardevOpts, ","))

	return append(args, chardevArg, devArg)
}

func fdPath(fd int) string {
	return fmt.Sprintf("/dev/fd/%d", fd)
}
