package podman

import (
	"arcaflow-engine-deployer-podman/util"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type cliWrapper struct {
	PodmanFullPath string
}

func NewCliWrapper(fullPath string) CliWrapper {
	return &cliWrapper{
		PodmanFullPath: fullPath,
	}
}

func (p *cliWrapper) decorateImageName(image string) string {
	imageParts := strings.Split(image, ":")
	if len(imageParts) == 1 {
		image = fmt.Sprintf("%s:latest", image)
	}
	return image
}

func (p *cliWrapper) commandSetEnv(command *[]string, env []string) {
	for _, v := range env {
		if tokens := strings.Split(v, "="); len(tokens) == 2 {
			*command = append(*command, "-e", v)
		}
	}
}

func (p *cliWrapper) commandSetVolumes(command *[]string, binds []string) {
	for _, v := range binds {
		if tokens := strings.Split(v, ":"); len(tokens) == 2 {
			*command = append(*command, "-v", v)
		}
	}
}

func (p *cliWrapper) commandSetCgroupNs(command *[]string, cgroupNs string) {
	if cgroupNs != "" {
		*command = append(*command, "--cgroupns", cgroupNs)
	}
}

func (p *cliWrapper) commandSetContainerName(command *[]string, name string) {
	if name != "" {
		*command = append(*command, "--name", name)
	}
}

func (p *cliWrapper) ImageExists(image string) (*bool, error) {
	image = p.decorateImageName(image)
	cmd := exec.Command(p.PodmanFullPath, "image", "ls", "--format", "{{.Repository}}:{{.Tag}}")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	outStr := out.String()
	outSlice := strings.Split(outStr, "\n")
	exists := util.SliceContains(outSlice, image)
	return &exists, nil
}

func (p *cliWrapper) PullImage(image string, platform *string) error {
	commandArgs := []string{"pull"}
	if platform != nil {
		commandArgs = append(commandArgs, []string{"--platform", *platform}...)
	}
	image = p.decorateImageName(image)
	commandArgs = append(commandArgs, image)
	cmd := exec.Command(p.PodmanFullPath, commandArgs...)
	var out bytes.Buffer
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return errors.New(out.String())
	}
	return nil
}

func (p *cliWrapper) Deploy(
	image string,
	containerName string,
	env []string,
	volumeBinds []string,
	cgroupNs string,
) (io.WriteCloser, io.ReadCloser, *exec.Cmd, error) {

	image = p.decorateImageName(image)
	commandArgs := []string{"run", "-i", "-a", "stdin", "-a", "stdout"}
	p.commandSetContainerName(&commandArgs, containerName)
	p.commandSetEnv(&commandArgs, env)
	p.commandSetVolumes(&commandArgs, volumeBinds)
	p.commandSetCgroupNs(&commandArgs, cgroupNs)
	commandArgs = append(commandArgs, image)
	cmd := exec.Command(p.PodmanFullPath, commandArgs...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, nil, errors.New(err.Error())
	}
	return stdin, stdout, cmd, nil
}
