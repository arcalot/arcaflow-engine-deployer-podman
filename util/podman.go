package util

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// SupportedPlatforms from: https://github.com/docker-library/official-images#architectures-other-than-amd64
func SupportedPlatforms() []string {
	return []string{
		"arm32v6",
		"arm32v7",
		"arm64v8",
		"amd64",
		"arm64",
		"linux/arm64",
		"winamd64",
		"arm32v5",
		"ppc64le",
		"s390x",
		"mips64le",
		"riscv64",
		"i386",
	}
}

type podman struct {
	PodmanFullPath string
}

type Podman interface {
	ImageExists(image string) (*bool, error)
	PullImage(image string, platform *string) error
}

func NewPodman(fullPath string) Podman {
	return &podman{
		PodmanFullPath: fullPath,
	}
}

func (p *podman) decorateImageName(image string) string {
	imageParts := strings.Split(image, ":")
	if len(imageParts) == 1 {
		image = fmt.Sprintf("%s:latest", image)
	}
	return image
}

func (p *podman) ImageExists(image string) (*bool, error) {
	image = p.decorateImageName(image)
	cmd := exec.Command(p.PodmanFullPath, "image", "ls", "--format", "{{.Repository}}:{{.Tag}}")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	outStr := out.String()
	outSlice := strings.Split(outStr, "\n")
	exists := SliceContains(outSlice, image)
	return &exists, nil
}

func (p *podman) PullImage(image string, platform *string) error {
	commandArgs := []string{"pull"}
	if platform != nil {
		if SliceContains(SupportedPlatforms(), *platform) == false {
			return errors.New(fmt.Sprintf("Unsupported platform: %s", *platform))
		}
		commandArgs = append(commandArgs, []string{"--platform", *platform}...)
	}
	image = p.decorateImageName(image)
	commandArgs = append(commandArgs, image)
	cmd := exec.Command(p.PodmanFullPath, commandArgs...)
	var out bytes.Buffer
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return errors.New(out.String())
	}
	return nil
}
