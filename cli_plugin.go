package podman

import (
	"fmt"
	"io"

	log "go.arcalot.io/log/v2"
	"go.flow.arcalot.io/podmandeployer/internal/cliwrapper"
)

type CliPlugin struct {
	wrapper        cliwrapper.CliWrapper
	containerImage string
	containerName  string
	config         *Config
	logger         log.Logger
	stdin          io.WriteCloser
	stdout         io.ReadCloser
}

func (p *CliPlugin) Write(b []byte) (n int, err error) {
	return p.stdin.Write(b)
}

func (p *CliPlugin) Read(b []byte) (n int, err error) {
	return p.stdout.Read(b)
}

func (p *CliPlugin) Close() error {
	containerRunning, err := p.wrapper.ContainerRunning(p.containerImage)
	if err != nil {
		p.logger.Warningf("error while checking if container exists (%s);"+
			" killing container in case it still exists", err.Error())
	} else if containerRunning {
		p.logger.Infof("container %s still exists; killing container", p.containerName)
	}
	var killErr error
	if err != nil || containerRunning {
		killErr = p.wrapper.Kill(p.containerName)
	}

	// Still clean up even if the kill fails.
	cleanErr := p.wrapper.Clean(p.containerName)

	if err := p.stdin.Close(); err != nil {
		p.logger.Warningf("failed to close stdin pipe")
	} else {
		p.logger.Debugf("stdin pipe successfully closed")
	}
	if err := p.stdout.Close(); err != nil {
		p.logger.Warningf("failed to close stdout pipe")
	} else {
		p.logger.Debugf("stdout pipe successfully closed")
	}
	switch {
	case killErr != nil && cleanErr != nil:
		return fmt.Errorf("error while killing pod (%s) and cleaning up pod (%s)", killErr.Error(), cleanErr.Error())
	case killErr != nil:
		return killErr
	case cleanErr != nil:
		return cleanErr
	}
	return nil
}

func (p *CliPlugin) ID() string {
	return p.containerName
}
