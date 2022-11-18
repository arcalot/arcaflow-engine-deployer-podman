package podman

import (
	"arcaflow-engine-deployer-podman/wrapper"
	"context"
	"go.arcalot.io/log"
	"go.flow.arcalot.io/deployer"
	"regexp"
	"strings"
	"sync"
)

type Connector struct {
	config *Config
	logger log.Logger
	podman wrapper.PodmanWrapper
}

var tagRegexp = regexp.MustCompile("^[a-zA-Z0-9.-]$")

func (c Connector) Deploy(ctx context.Context, image string) (deployer.Plugin, error) {
	if err := c.pullImage(ctx, image); err != nil {
		return nil, err
	}
	podmanWrapper := wrapper.NewPodmanWrapper(c.config.Podman.Path)
	podmanConnector := PodmanConnector{
		ContainerOut: []byte{},
		Wrapper:      podmanWrapper,
		Lock:         &sync.Mutex{},
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

		parts := strings.Split(image, ":")
		tag := parts[len(parts)-1]
		if len(parts) > 1 && tagRegexp.MatchString(tag) && tag != "latest" && *imageExists {
			return nil
		}
	}
	return nil
}
