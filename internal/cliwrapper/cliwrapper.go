package cliwrapper

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	log "go.arcalot.io/log/v2"
	"go.flow.arcalot.io/podmandeployer/internal/util"
)

type cliWrapper struct {
	podmanFullPath string
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

func (p *cliWrapper) ImageExists(image string) (*bool, error) {
	image = p.decorateImageName(image)
	cmd := exec.Command(p.podmanFullPath, "image", "ls", "--format", "{{.Repository}}:{{.Tag}}") //nolint:gosec
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	p.logger.Debugf("Checking whether image exists with command %v", cmd.Args)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf(
			"error while determining if image exists. Stdout: '%s', Stderr: '%s', Cmd error: '%s'",
			out.String(), errOut.String(), err)
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
	cmd := exec.Command(p.podmanFullPath, commandArgs...) //nolint:gosec
	p.logger.Debugf("Pulling image with command %v", cmd.Args)
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"error while pulling image. Stdout: '%s', Stderr: '%s', Cmd error: '%s'",
			out.String(), errOut.String(), err)
	}
	return nil
}

func (p *cliWrapper) Deploy(image string, podmanArgs []string, containerArgs []string) (io.WriteCloser, io.ReadCloser, error) {
	image = p.decorateImageName(image)
	podmanArgs = append(podmanArgs, image)
	podmanArgs = append(podmanArgs, containerArgs...)
	deployCommand := exec.Command(p.podmanFullPath, podmanArgs...) //nolint:gosec
	p.logger.Debugf("Deploying with command %v", deployCommand.Args)
	stdin, err := deployCommand.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	stdout, err := deployCommand.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := deployCommand.Start(); err != nil {
		return nil, nil, errors.New(err.Error())
	}
	return stdin, stdout, nil
}

func (p *cliWrapper) KillAndClean(containerName string) error {
	cmdKill := exec.Command(p.podmanFullPath, "kill", containerName) //nolint:gosec
	p.logger.Debugf("Killing with command %v", cmdKill.Args)
	if err := cmdKill.Run(); err != nil {
		p.logger.Warningf("failed to kill pod %s, probably the execution terminated earlier", containerName)
	} else {
		p.logger.Warningf("successfully killed container %s", containerName)
	}

	var cmdRmContainerStderr bytes.Buffer
	cmdRmContainer := exec.Command(p.podmanFullPath, "rm", "--force", containerName) //nolint:gosec
	p.logger.Debugf("Removing container with command %v", cmdRmContainer.Args)
	cmdRmContainer.Stderr = &cmdRmContainerStderr
	if err := cmdRmContainer.Run(); err != nil {
		p.logger.Errorf("failed to remove container %s: %s", containerName, cmdRmContainerStderr.String())
	} else {
		p.logger.Infof("successfully removed container %s", containerName)
	}
	return nil
}
