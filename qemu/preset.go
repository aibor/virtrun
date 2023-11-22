package qemu

import (
	"fmt"
	"os"
	"runtime"
)

var CommandPresets = map[string]Command{
	"amd64": {
		Binary:        "qemu-system-x86_64",
		Machine:       "q35",
		TransportType: TransportTypePCI,
		CPU:           "max",
		ExtraArgs: Arguments{
			ArgDisplay("none"),
			ArgMonitor("none"),
			UniqueArg("no-reboot"),
			UniqueArg("nodefaults"),
			UniqueArg("no-user-config"),
		},
	},
	"arm64": {
		Binary:        "qemu-system-aarch64",
		Machine:       "virt",
		TransportType: TransportTypeMMIO,
		CPU:           "max",
		ExtraArgs: Arguments{
			ArgDisplay("none"),
			ArgMonitor("none"),
			UniqueArg("no-reboot"),
			UniqueArg("nodefaults"),
			UniqueArg("no-user-config"),
		},
	},
}

// CommandFor creates a new [qemu.Command] with defaults set to the given
// architecture. If it does not match the host architecture, the
// [Command.NoKVM] flag ist set. Supported architectures so far: amd64, arm64.
func CommandFor(arch string) (Command, error) {
	cmd, exists := CommandPresets[arch]
	if !exists {
		return Command{}, fmt.Errorf("arch not supported: %s", arch)
	}

	cmd.Memory = 256
	cmd.SMP = 1
	cmd.NoKVM = !KVMAvailableFor(arch)

	return cmd, nil
}

// KVMAvailableFor checks if KVM support is available for the given
// architecture.
func KVMAvailableFor(arch string) bool {
	if runtime.GOARCH != arch {
		return false
	}
	f, err := os.OpenFile("/dev/kvm", os.O_WRONLY, 0)
	_ = f.Close()
	return err == nil
}
