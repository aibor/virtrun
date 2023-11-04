package qemu

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type Argument struct {
	Name          string
	Value         string
	NonUniqueName bool
}

func (a *Argument) Equal(b Argument) bool {
	if a.Name != b.Name {
		return false
	}
	if a.NonUniqueName {
		return a.Value == b.Value
	}
	return true
}

func (a Argument) WithValue() func(string) Argument {
	return func(s string) Argument {
		a.Value = s
		return a
	}
}

func (a Argument) WithMultiValue(separator string) func(...string) Argument {
	if separator == "" {
		separator = ","
	}
	return func(s ...string) Argument {
		return a.WithValue()(strings.Join(s, separator))
	}
}

func (a Argument) WithIntValue() func(int) Argument {
	return func(i int) Argument {
		return a.WithValue()(strconv.Itoa(i))
	}
}

func UniqueArg(name string) Argument {
	return Argument{
		Name: name,
	}
}

func RepeatableArg(name string) Argument {
	return Argument{
		Name:          name,
		NonUniqueName: true,
	}
}

var (
	ArgKernel  = UniqueArg("kernel").WithValue()
	ArgInitrd  = UniqueArg("initrd").WithValue()
	ArgMachine = UniqueArg("machine").WithValue()
	ArgCPU     = UniqueArg("cpu").WithValue()
	ArgSMP     = UniqueArg("smp").WithIntValue()
	ArgMemory  = UniqueArg("m").WithIntValue()
	ArgDisplay = UniqueArg("display").WithValue()
	ArgMonitor = UniqueArg("monitor").WithValue()
	ArgSerial  = RepeatableArg("serial").WithValue()
	ArgDevice  = RepeatableArg("device").WithMultiValue("")
	ArgChardev = RepeatableArg("chardev").WithMultiValue("")
	ArgAppend  = RepeatableArg("append").WithMultiValue(" ")
)

type Arguments []Argument

func (a *Arguments) Add(e ...Argument) {
	*a = append(*a, e...)
}

func (a Arguments) Build() ([]string, error) {
	s := make([]string, 0, len(a))
	for idx, e := range a {
		if slices.ContainsFunc(a[idx+1:], e.Equal) {
			return nil, fmt.Errorf("colliding args: %s", e.Name)
		}
		s = append(s, "-"+e.Name)
		if e.Value != "" {
			s = append(s, e.Value)
		}
	}
	return s, nil
}
