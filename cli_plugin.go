package podman

import (
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
	containerExists, err := p.wrapper.ContainerExists(p.containerImage)
	if err != nil {
		p.logger.Warningf("error while checking if container exists (%s);"+
			" killing container in case it still exists", err.Error())
	} else if containerExists {
		p.logger.Infof("container %s still exists; killing container", p.containerName)
	}
	if err != nil || containerExists {
		if err := p.wrapper.KillAndClean(p.containerName); err != nil {
			return err
		}
	}

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
	return nil
}

func (p *CliPlugin) ID() string {
	return p.containerName
}
