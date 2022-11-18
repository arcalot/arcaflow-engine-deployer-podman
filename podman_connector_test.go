package podman

import (
	"arcaflow-engine-deployer-podman/wrapper"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"sync"
	"testing"
)

func TestPodmanConnector_Write(t *testing.T) {
	config := Config{
		Podman: Podman{
			Path: GetPodmanPath(),
		},
		Deployment: Deployment{
			ContainerConfig: &container.Config{
				Image: testImageIo,
			},
		},
	}
	podmanWrapper := wrapper.NewPodmanWrapper(GetPodmanPath())

	podmanConnector := PodmanConnector{
		Config:       &config,
		ContainerOut: []byte{},
		Lock:         &sync.Mutex{},
		Wrapper:      podmanWrapper,
	}
	command := "ls -al"
	podmanConnector.Write([]byte(command))
	var out []byte
	podmanConnector.Read(out)
	fmt.Print(string(out))
}
