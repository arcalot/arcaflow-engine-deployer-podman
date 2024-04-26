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
	runtimeContext []string
}

func NewCliWrapper(fullPath string, logger log.Logger, context string) CliWrapper {
	// Specify podman --connection string if provided
	if context != "" {
		context = "--connection=" + context
	}
	return &cliWrapper{
		podmanFullPath: fullPath,
		logger:         logger,
		runtimeContext: []string{context},
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
	cmd := p.getPodmanCmd("image", "ls", "--format", "{{.Repository}}:{{.Tag}}")
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	p.logger.Debugf("Checking whether image exists with command %v", cmd.Args)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf(
			"error while determining if image exists. Stdout: '%s', Stderr: '%s', Cmd error: (%w)",
			out.String(), errOut.String(), err)
	}
	outStr := out.String()
	outSlice := strings.Split(outStr, "\n")
	exists := util.SliceContains(outSlice, p.decorateImageName(image))
	return &exists, nil
}

func (p *cliWrapper) PullImage(image string, platform *string) error {
	commandArgs := []string{"pull"}
	if platform != nil {
		commandArgs = append(commandArgs, "--platform", *platform)
	}
	commandArgs = append(commandArgs, p.decorateImageName(image))
	cmd := p.getPodmanCmd(commandArgs...)
	p.logger.Debugf("Pulling image with command %v", cmd.Args)
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"error while pulling image. Stdout: '%s', Stderr: '%s', Cmd error: (%w)",
			out.String(), errOut.String(), err)
	}
	return nil
}

func (p *cliWrapper) Deploy(image string, podmanArgs []string, containerArgs []string) (io.WriteCloser, io.ReadCloser, error) {
	podmanArgs = append(podmanArgs, p.decorateImageName(image))
	podmanArgs = append(podmanArgs, containerArgs...)
	deployCommand := p.getPodmanCmd(podmanArgs...)
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
	cmdKill := p.getPodmanCmd("kill", containerName)
	p.logger.Debugf("Killing with command %v", cmdKill.Args)
	if err := cmdKill.Run(); err != nil {
		p.logger.Warningf("failed to kill pod %s, probably the execution terminated earlier", containerName)
	} else {
		p.logger.Warningf("successfully killed container %s", containerName)
	}

	var cmdRmContainerStderr bytes.Buffer
	cmdRmContainer := p.getPodmanCmd("rm", "--force", containerName)
	p.logger.Debugf("Removing container with command %v", cmdRmContainer.Args)
	cmdRmContainer.Stderr = &cmdRmContainerStderr
	if err := cmdRmContainer.Run(); err != nil {
		p.logger.Errorf("failed to remove container %s: %s", containerName, cmdRmContainerStderr.String())
	} else {
		p.logger.Infof("successfully removed container %s", containerName)
	}
	return nil
}

func (p *cliWrapper) getPodmanCmd(cmdArgs ...string) *exec.Cmd {
	commandArgs := p.runtimeContext
	commandArgs = append(commandArgs, cmdArgs...)
	return exec.Command(p.podmanFullPath, commandArgs...) //#nosec G204 -- command line is internally generated
}
