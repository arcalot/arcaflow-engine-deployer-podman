package argsbuilder

import (
	"strings"
)

type argsBuilder struct {
	commandArgs *[]string
}

func (a *argsBuilder) SetEnv(env []string) ArgsBuilder {
	for _, v := range env {
		if tokens := strings.Split(v, "="); len(tokens) == 2 {
			*a.commandArgs = append(*a.commandArgs, "-e", v)
		}
	}
	return a
}

func (a *argsBuilder) SetVolumes(binds []string) ArgsBuilder {
	for _, v := range binds {
		if tokens := strings.Split(v, ":"); len(tokens) == 2 {
			*a.commandArgs = append(*a.commandArgs, "-v", v)
		}
	}
	return a
}

func (a *argsBuilder) SetCgroupNs(cgroupNs string) ArgsBuilder {
	if cgroupNs != "" {
		*a.commandArgs = append(*a.commandArgs, "--cgroupns", cgroupNs)
	}
	return a
}

func (a *argsBuilder) SetContainerName(name string) ArgsBuilder {
	if name != "" {
		*a.commandArgs = append(*a.commandArgs, "--name", name)
	}
	return a
}

func (a *argsBuilder) SetNetworkMode(networkMode string) ArgsBuilder {
	if networkMode != "" {
		*a.commandArgs = append(*a.commandArgs, "--network", networkMode)
	}
	return a
}
