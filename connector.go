package podman

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	log "go.arcalot.io/log/v2"
	"go.flow.arcalot.io/deployer"
	args "go.flow.arcalot.io/podmandeployer/internal/argsbuilder"
	"go.flow.arcalot.io/podmandeployer/internal/cliwrapper"
	"go.flow.arcalot.io/podmandeployer/internal/util"
	"math/rand"
)

type Connector struct {
	containerNameRoot string
	config            *Config
	logger            log.Logger
	podmanCliWrapper  cliwrapper.CliWrapper
	rng               *rand.Rand
	seed              int64
}

func (c *Connector) Deploy(ctx context.Context, image string) (deployer.Plugin, error) {
	if err := c.pullImage(ctx, image); err != nil {
		return nil, err
	}
	if c.config.Podman.Path == "" {
		c.logger.Errorf("oops, neither podman -> path provided in configuration nor binary found in $PATH")
		panic("oops, neither podman -> path provided in configuration nor binary found in $PATH")
	}

	containerConfig := c.unwrapContainerConfig()
	hostConfig := c.unwrapHostConfig()
	commandArgs := []string{"run", "-i", "-a", "stdin", "-a", "stdout", "-a", "stderr"}
	containerName := c.NextContainerName(c.containerNameRoot, 10)

	args.NewBuilder(&commandArgs).
		SetContainerName(containerName).
		SetEnv(containerConfig.Env).
		SetVolumes(hostConfig.Binds).
		SetCgroupNs(c.config.Podman.CgroupNs).
		SetNetworkMode(c.config.Podman.NetworkMode)

	stdin, stdout, err := c.podmanCliWrapper.Deploy(image, commandArgs, []string{"--atp"})

	if err != nil {
		return nil, err
	}

	cliPlugin := CliPlugin{
		wrapper:        c.podmanCliWrapper,
		containerImage: image,
		containerName:  containerName,
		config:         c.config,
		stdin:          stdin,
		stdout:         stdout,
		logger:         c.logger,
	}

	return &cliPlugin, nil
}

func (c *Connector) pullImage(_ context.Context, image string) error {
	if c.config.Deployment.ImagePullPolicy == ImagePullPolicyNever {
		return nil
	}
	if c.config.Deployment.ImagePullPolicy == ImagePullPolicyIfNotPresent {
		imageExists, err := c.podmanCliWrapper.ImageExists(image)
		if err != nil {
			return err
		}

		if *imageExists {
			c.logger.Debugf("%s: image already present skipping pull", image)
			return nil
		}
		// TODO:fix default values in configuration

		c.logger.Debugf("Pulling image: %s", image)
		if err := c.podmanCliWrapper.PullImage(image, &c.config.Podman.ImageArchitecture); err != nil {
			return err
		}
	}
	return nil
}

func (c *Connector) unwrapContainerConfig() container.Config {
	if c.config.Deployment.ContainerConfig != nil {
		return *c.config.Deployment.ContainerConfig
	}
	return container.Config{}
}

func (c *Connector) unwrapHostConfig() container.HostConfig {
	if c.config.Deployment.HostConfig != nil {
		return *c.config.Deployment.HostConfig
	}
	return container.HostConfig{}
}

func (c *Connector) NextContainerName(container_id string, random_str_size int) string {
	return fmt.Sprintf("%s_%s", container_id, util.GetRandomString(c.rng, random_str_size))
}
