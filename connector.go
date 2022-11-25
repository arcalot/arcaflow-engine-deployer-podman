package podman

import (
	args "arcaflow-engine-deployer-podman/internal/args_builder"
	cli_wrapper2 "arcaflow-engine-deployer-podman/internal/cli_wrapper"
	"context"
	"github.com/docker/docker/api/types/container"
	"go.arcalot.io/log"
	"go.flow.arcalot.io/deployer"
	"regexp"
	"sync"
)

type Connector struct {
	config *Config
	logger log.Logger
	podman cli_wrapper2.CliWrapper
}

var tagRegexp = regexp.MustCompile("^[a-zA-Z0-9.-]$")

func (c Connector) Deploy(ctx context.Context, image string) (deployer.Plugin, error) {
	if err := c.pullImage(ctx, image); err != nil {
		return nil, err
	}
	if c.config.Podman.Path == "" {
		c.logger.Errorf("oops, neither podman -> path provided in configuration nor binary found in $PATH")
		panic("oops, neither podman -> path provided in configuration nor binary found in $PATH")
	}
	cliWrapper := cli_wrapper2.NewCliWrapper(c.config.Podman.Path)

	containerConfig := c.unwrapContainerConfig()
	hostConfig := c.unwrapHostConfig()
	commandArgs := []string{"run", "-i", "-a", "stdin", "-a", "stdout", "-a", "stderr"}
	args.NewBuilder(&commandArgs).
		SetContainerName(c.config.Podman.ContainerName).
		SetEnv(containerConfig.Env).
		SetVolumes(hostConfig.Binds).
		SetCgroupNs(c.config.Podman.CgroupNs)
	stdin, stdout, _, _, err := cliWrapper.Deploy(image, image, commandArgs)

	if err != nil {
		return nil, err
	}

	cliPlugin := CliPlugin{
		wrapper:        cliWrapper,
		lock:           &sync.Mutex{},
		containerImage: image,
		config:         c.config,
		stdin:          stdin,
		stdout:         stdout,
	}

	return &cliPlugin, nil
}

func (c *Connector) pullImage(ctx context.Context, image string) error {
	if c.config.Deployment.ImagePullPolicy == ImagePullPolicyNever {
		return nil
	}
	if c.config.Deployment.ImagePullPolicy == ImagePullPolicyIfNotPresent {
		imageExists, err := c.podman.ImageExists(image)
		if err != nil {
			return err
		}

		if *imageExists {
			c.logger.Debugf("%s: image already present skipping pull", image)
			return nil
		}
		//TODO:fix default values in configuration
		_amd64 := "amd64"
		c.logger.Debugf("Pulling image: %s", image)
		if err := c.podman.PullImage(image, &_amd64); err != nil {
			return err
		}
	}
	return nil
}

func (c *Connector) unwrapContainerConfig() container.Config {
	if c.config.Deployment.ContainerConfig != nil {
		return *c.config.Deployment.ContainerConfig
	} else {
		return container.Config{}
	}
}

func (c *Connector) unwrapHostConfig() container.HostConfig {
	if c.config.Deployment.HostConfig != nil {
		return *c.config.Deployment.HostConfig
	} else {
		return container.HostConfig{}
	}
}
