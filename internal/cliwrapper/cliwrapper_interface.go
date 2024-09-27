package cliwrapper

import (
	"io"
)

type CliWrapper interface {
	ImageExists(image string) (*bool, error)
	ContainerExists(image string) (bool, error)
	PullImage(image string, platform *string) error
	Deploy(
		image string,
		podmanArgs []string,
		containerArgs []string,
	) (io.WriteCloser, io.ReadCloser, error)
	KillAndClean(containerName string) error
}
