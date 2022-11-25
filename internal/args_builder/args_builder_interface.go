package args_builder

type ArgsBuilder interface {
	SetEnv(env []string) ArgsBuilder
	SetVolumes(binds []string) ArgsBuilder
	SetCgroupNs(cgroupNs string) ArgsBuilder
	SetContainerName(name string) ArgsBuilder
	SetPort(port string) ArgsBuilder
}

func NewBuilder(commandArgs *[]string) ArgsBuilder {
	return &argsBuilder{
		commandArgs: commandArgs,
	}
}
