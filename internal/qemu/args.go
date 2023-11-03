package qemu

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type arg struct {
	name          string
	value         string
	nonUniqueName bool
}

func (a *arg) equal(b arg) bool {
	if a.name != b.name {
		return false
	}
	if a.nonUniqueName {
		return a.value == b.value
	}
	return true
}

func (a arg) withValue() func(string) arg {
	return func(s string) arg {
		a.value = s
		return a
	}
}

func (a arg) withMultiValue(separator string) func(...string) arg {
	if separator == "" {
		separator = ","
	}
	return func(s ...string) arg {
		return a.withValue()(strings.Join(s, separator))
	}
}

func (a arg) withIntValue() func(int) arg {
	return func(i int) arg {
		return a.withValue()(strconv.Itoa(i))
	}
}

func uniqueArg(name string) arg {
	return arg{
		name: name,
	}
}

func repeatableArg(name string) arg {
	return arg{
		name:          name,
		nonUniqueName: true,
	}
}

var (
	argKernel  = uniqueArg("kernel").withValue()
	argInitrd  = uniqueArg("initrd").withValue()
	argMachine = uniqueArg("machine").withValue()
	argCPU     = uniqueArg("cpu").withValue()
	argSMP     = uniqueArg("smp").withIntValue()
	argMemory  = uniqueArg("m").withIntValue()
	argDisplay = uniqueArg("display").withValue()
	argMonitor = uniqueArg("monitor").withValue()
	argSerial  = repeatableArg("serial").withValue()
	argDevice  = repeatableArg("device").withMultiValue("")
	argChardev = repeatableArg("chardev").withMultiValue("")
	argAppend  = repeatableArg("append").withMultiValue(" ")
)

type args []arg

func (a *args) build() ([]string, error) {
	s := make([]string, 0, len(*a))
	for idx, e := range *a {
		if slices.ContainsFunc((*a)[idx+1:], e.equal) {
			return nil, fmt.Errorf("colliding args: %s", e.name)
		}
		s = append(s, "-"+e.name)
		if e.value != "" {
			s = append(s, e.value)
		}
	}
	return s, nil
}
