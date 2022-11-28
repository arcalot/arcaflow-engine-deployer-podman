package podman

import (
	"arcaflow-engine-deployer-podman/internal/cliwrapper"
	"go.arcalot.io/log"
	"io"
)

type CliPlugin struct {
	wrapper        cliwrapper.CliWrapper
	containerImage string
	containerName  string
	readIndex      int64
	config         *Config
	logger         log.Logger
	stdin          io.WriteCloser
	stdout         io.ReadCloser
}

// TODO: unwrap the whole config

func (p *CliPlugin) Write(b []byte) (n int, err error) {
	return p.stdin.Write(b)
}

func (p *CliPlugin) Read(b []byte) (n int, err error) {
	return p.stdout.Read(b)
}

func (p *CliPlugin) Close() error {
	if err := p.wrapper.KillAndWait(p.containerName); err != nil {
		return err
	}
	return nil
}
