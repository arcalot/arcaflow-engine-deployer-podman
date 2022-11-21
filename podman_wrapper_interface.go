package podman

import (
	"io"
	"os/exec"
)

type PodmanWrapper interface {
	ImageExists(image string) (*bool, error)
	PullImage(image string, platform *string) error
	Deploy(image string) (io.WriteCloser, io.ReadCloser, *exec.Cmd, error)
}
