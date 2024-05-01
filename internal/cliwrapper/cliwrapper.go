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
	connectionName []string
}

func NewCliWrapper(fullPath string, logger log.Logger, connectionName string) CliWrapper {
	// Specify podman --connection string if provided
	connection := []string{}
	if connectionName != "" {
		connection = []string{"--connection=" + connectionName}
	}
	logger.Debugf("ConnectionName %v %v", connectionName, connection)

	return &cliWrapper{
		podmanFullPath: fullPath,
		logger:         logger,
		connectionName: connection,
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
	outStr, err := p.runPodmanCmd(
		"checking whether image exists",
		"image", "ls", "--format", "{{.Repository}}:{{.Tag}}",
	)
	if err != nil {
		return nil, err
	}
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
	_, err := p.runPodmanCmd("pulling image", commandArgs...)
	return err
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

	msg := "removing container " + containerName
	_, err := p.runPodmanCmd(msg, "rm", "--force", containerName)
	if err != nil {
		p.logger.Errorf(err.Error())
	} else {
		p.logger.Infof("successfully removed container %s", containerName)
	}
	return nil
}

func (p *cliWrapper) getPodmanCmd(cmdArgs ...string) *exec.Cmd {
	var commandArgs []string
	commandArgs = append(commandArgs, p.connectionName...)
	commandArgs = append(commandArgs, cmdArgs...)
	return exec.Command(p.podmanFullPath, commandArgs...) //#nosec G204 -- command line is internally generated
}

func (p *cliWrapper) runPodmanCmd(msg string, cmdArgs ...string) (string, error) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	cmd := p.getPodmanCmd(cmdArgs...)
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	p.logger.Debugf(msg+" with command %v", cmd.Args)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf(
			"error while %s. Stdout: '%s', Stderr: '%s', Cmd error: (%w)",
			msg, strings.TrimSpace(out.String()), strings.TrimSpace(errOut.String()), err)
	}
	return out.String(), nil
}
