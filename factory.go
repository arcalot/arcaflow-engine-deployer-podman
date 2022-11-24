package podman

import (
	"arcaflow-engine-deployer-podman/cli_wrapper"
	"arcaflow-engine-deployer-podman/config"
	"go.arcalot.io/log"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/pluginsdk/schema"
)

// NewFactory creates a new factory for the Docker deployer.
func NewFactory() deployer.ConnectorFactory[*config.Config] {
	return &factory{}
}

type factory struct {
}

func (f factory) ID() string {
	return "docker"
}

func (f factory) ConfigurationSchema() *schema.TypedScopeSchema[*config.Config] {
	return config.Schema
}

func (f factory) Create(config *config.Config, logger log.Logger) (deployer.Connector, error) {
	podman := cli_wrapper.NewCliWrapper(config.Podman.Path)
	return Connector{
		config: config,
		logger: logger,
		podman: podman,
	}, nil
}
