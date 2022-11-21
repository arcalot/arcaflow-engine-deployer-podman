package podman

import (
	"context"
	"go.arcalot.io/log"
	"go.flow.arcalot.io/deployer"
	"regexp"
	"sync"
)

type Connector struct {
	config *Config
	logger log.Logger
	podman PodmanWrapper
}

var tagRegexp = regexp.MustCompile("^[a-zA-Z0-9.-]$")

func (c Connector) Deploy(ctx context.Context, image string) (deployer.Plugin, error) {
	if err := c.pullImage(ctx, image); err != nil {
		return nil, err
	}

	podmanWrapper := NewPodmanWrapper("/usr/bin/podman")
	podmanConnector := Cli{
		wrapper:        podmanWrapper,
		lock:           &sync.Mutex{},
		containerImage: image,
	}
	return &podmanConnector, nil
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
