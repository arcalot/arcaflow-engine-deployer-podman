package podman

import (
	"fmt"

	"go.arcalot.io/log"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/pluginsdk/schema"
	"go.flow.arcalot.io/podmandeployer/internal/cliwrapper"
	"go.flow.arcalot.io/podmandeployer/internal/util"
)

// NewFactory creates a new factory for the Docker deployer.
func NewFactory() deployer.ConnectorFactory[*Config] {
	return &factory{}
}

type factory struct {
}

func (f factory) ID() string {
	return "podman"
}

func (f factory) ConfigurationSchema() *schema.TypedScopeSchema[*Config] {
	return Schema
}

func (f factory) Create(config *Config, logger log.Logger) (deployer.Connector, error) {
	podman := cliwrapper.NewCliWrapper(config.Podman.Path, logger)
	var containerName string
	if config.Podman.ContainerName == "" {
		containerName = fmt.Sprintf("arcaflow_podman_%s", util.GetRandomString(5))
	} else {
		containerName = config.Podman.ContainerName
	}
	return &Connector{
		config:           config,
		logger:           logger,
		podmanCliWrapper: podman,
		containerName:    containerName,
	}, nil
}
