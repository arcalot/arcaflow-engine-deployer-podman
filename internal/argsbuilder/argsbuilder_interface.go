package argsbuilder

type ArgsBuilder interface {
	SetEnv(env []string) ArgsBuilder
	SetVolumes(binds []string) ArgsBuilder
	SetCgroupNs(cgroupNs string) ArgsBuilder
	SetContainerName(name string) ArgsBuilder
	SetNetworkMode(networkMode string) ArgsBuilder
	SetPrivileged(privileged bool) ArgsBuilder
}

func NewBuilder(commandArgs *[]string) ArgsBuilder {
	return &argsBuilder{
		commandArgs: commandArgs,
	}
}
