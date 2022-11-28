package cliwrapper

import (
	"arcaflow-engine-deployer-podman/internal/util"
	"bytes"
	"errors"
	"fmt"
	"go.arcalot.io/log"
	"io"
	"os/exec"
	"strings"
)

type cliWrapper struct {
	podmanFullPath string
	deployCommand  *exec.Cmd
	logger         log.Logger
}

func NewCliWrapper(fullPath string, logger log.Logger) CliWrapper {
	return &cliWrapper{
		podmanFullPath: fullPath,
		logger:         logger,
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
	cmd := exec.Command(p.podmanFullPath, "image", "ls", "--format", "{{.Repository}}:{{.Tag}}")
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
	cmd := exec.Command(p.podmanFullPath, commandArgs...)
	var out bytes.Buffer
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return errors.New(out.String())
	}
	return nil
}

func (p *cliWrapper) Deploy(image string, args []string) (io.WriteCloser, io.ReadCloser, error) {
	image = p.decorateImageName(image)
	args = append(args, image)
	p.deployCommand = exec.Command(p.podmanFullPath, args...)
	stdin, err := p.deployCommand.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	stdout, err := p.deployCommand.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := p.deployCommand.Start(); err != nil {
		return nil, nil, errors.New(err.Error())
	}
	return stdin, stdout, nil
}

func (p *cliWrapper) KillAndWait(containerName string) error {

	if p.deployCommand != nil {
		cmdKill := exec.Command(p.podmanFullPath, "kill", containerName)
		if err := cmdKill.Run(); err != nil {
			p.logger.Warningf("failed to kill pod %s, probably the execution terminated earlier", containerName)
		}
		if err := p.deployCommand.Wait(); err != nil {
			p.logger.Warningf("failed to wait for pod %s, probably the execution terminated and pipes closed.", containerName)
		}
		var cmdRmContainerStderr bytes.Buffer
		cmdRmContainer := exec.Command(p.podmanFullPath, "rm", "--force", containerName)
		cmdRmContainer.Stderr = &cmdRmContainerStderr
		if err := cmdRmContainer.Run(); err != nil {
			p.logger.Errorf("failed to remove container %s: %s", containerName, cmdRmContainerStderr.String())
		} else {
			p.logger.Infof("successfully removed container %s", containerName)
		}
	}
	return nil
}
