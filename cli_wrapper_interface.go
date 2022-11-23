package podman

import (
	"io"
	"os/exec"
)

type CliWrapper interface {
	ImageExists(image string) (*bool, error)
	PullImage(image string, platform *string) error
	Deploy(
		image string,
		containerName string,
		env []string,
		volumeBinds []string,
		cgroupNs string,
	) (io.WriteCloser, io.ReadCloser, *exec.Cmd, error)
}
