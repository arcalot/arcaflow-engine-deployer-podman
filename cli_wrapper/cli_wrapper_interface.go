package cli_wrapper

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
		args []string,
	) (io.WriteCloser, io.ReadCloser, io.ReadCloser, *exec.Cmd, error)
}
