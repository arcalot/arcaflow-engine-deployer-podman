package podman

import (
	"arcaflow-engine-deployer-podman/util"
	"context"
	"go.arcalot.io/log"
	"go.flow.arcalot.io/deployer"
	"os/exec"
	"regexp"
	"strings"
)

type Connector struct {
	config *Config
	logger log.Logger
	podman util.Podman
}

var tagRegexp = regexp.MustCompile("^[a-zA-Z0-9.-]$")

func (c *Connector) Deploy(ctx context.Context, image string) (deployer.Plugin, error) {
	if err := c.pullImage(ctx, image); err != nil {
		return nil, err
	}
}

func (c *Connector) pullImage(ctx context.Context, image string) error {
	if c.config.Deployment.ImagePullPolicy == ImagePullPolicyNever {
		return nil
	}
	if c.config.Deployment.ImagePullPolicy == ImagePullPolicyIfNotPresent {

		exec.Command(c.config.Podman.Path, "image", "ls", "--format", "{{.Repository}}:{{.Tag}}")

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
}
