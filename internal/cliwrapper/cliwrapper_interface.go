package cliwrapper

import (
	"io"
)

type CliWrapper interface {
	ImageExists(image string) (*bool, error)
	PullImage(image string, platform *string) error
	Deploy(
		image string,
		args []string,
	) (io.WriteCloser, io.ReadCloser, error)
	KillAndWait(containerName string) error
}
