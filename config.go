package podman

import (
	"github.com/docker/docker/api/types/container"
	"time"
)

type Config struct {
	Podman     Podman     `json:"podman"`
	Deployment Deployment `json:"deployment"`
	Timeouts   Timeouts   `json:"timeouts"`
}

// Validate checks the configuration structure for conformance with the schema.
func (c *Config) Validate() error {
	return Schema.ValidateType(c)
}

// ImagePullPolicy drives when an image should be pulled.
type ImagePullPolicy string

const (
	// ImagePullPolicyAlways means that the container image will be pulled for every workflow run.
	ImagePullPolicyAlways ImagePullPolicy = "Always"
	// ImagePullPolicyIfNotPresent means the image will be pulled if the image is not present locally, an empty tag, or
	// the "latest" tag was specified.
	ImagePullPolicyIfNotPresent ImagePullPolicy = "IfNotPresent"
	// ImagePullPolicyNever means that the image will never be pulled, and if the image is not available locally the
	// execution will fail.
	ImagePullPolicyNever ImagePullPolicy = "Never"
)

type Podman struct {
	Path                string `json:"path"`
	ContainerNamePrefix string `json:"containerNamePrefix"`
	CgroupNs            string `json:"cgroupNs"`
	NetworkMode         string `json:"networkMode"`
	ImageArchitecture   string `json:"imageArchitecture"`
	ImageOS             string `json:"imageOS"`
	RngSeed             int64  `json:"rngSeed"`
}

// Deployment contains the information about deploying the plugin.
type Deployment struct {
	ContainerConfig *container.Config     `json:"container"`
	HostConfig      *container.HostConfig `json:"host"`
	ImagePullPolicy ImagePullPolicy       `json:"imagePullPolicy"`
}

// Timeouts drive the timeouts for various interactions in relation to Docker.
type Timeouts struct {
	HTTP time.Duration `json:"http"`
}
