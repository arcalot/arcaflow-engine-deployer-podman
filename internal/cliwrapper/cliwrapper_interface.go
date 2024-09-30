package cliwrapper

import (
	"io"
)

type CliWrapper interface {
	ImageExists(image string) (*bool, error)
	ContainerRunning(image string) (bool, error)
	PullImage(image string, platform *string) error
	Deploy(
		image string,
		podmanArgs []string,
		containerArgs []string,
	) (io.WriteCloser, io.ReadCloser, error)
	Kill(containerName string) error
	Clean(containerName string) error
}
