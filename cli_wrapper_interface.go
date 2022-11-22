package podman

import (
	"io"
	"os/exec"
)

type CliWrapper interface {
	ImageExists(image string) (*bool, error)
	PullImage(image string, platform *string) error
	Deploy(image string, env *[]string) (io.WriteCloser, io.ReadCloser, *exec.Cmd, error)
}
