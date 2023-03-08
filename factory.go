package podman

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	log "go.arcalot.io/log/v2"
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
	podmanPath, err := binaryCheck(config.Podman.Path)
	if err != nil {
		return &Connector{}, fmt.Errorf("podman binary check failed with error: %w", err)
	}
	podman := cliwrapper.NewCliWrapper(podmanPath, logger)
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

func binaryCheck(podmanPath string) (string, error) {
	if podmanPath == "" {
		podmanPath = "podman"
	}
	if !filepath.IsAbs(podmanPath) {
		podmanPathAbs, err := exec.LookPath(podmanPath)
		if err != nil {
			return "", fmt.Errorf("podman executable not found in a valid path with error: %w", err)

		}
		podmanPath = podmanPathAbs
	}
	if _, err := os.Stat(podmanPath); err != nil {
		return "", fmt.Errorf("podman binary not found with error: %w", err)
	}
	return podmanPath, nil
}
