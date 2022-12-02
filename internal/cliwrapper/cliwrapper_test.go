package cliwrapper

import (
	"fmt"
	"os/exec"
	"testing"

	"go.arcalot.io/log"

	"go.arcalot.io/assert"
	"go.flow.arcalot.io/podmandeployer/tests"
)

func TestPodman_ImageExists(t *testing.T) {
	logger := log.NewTestLogger(t)
	tests.RemoveImage(logger, tests.TestImage)

	podman := NewCliWrapper(tests.GetPodmanPath(), logger)

	assert.NotNil(t, tests.GetPodmanPath())

	cmd := exec.Command(tests.GetPodmanPath(), "pull", tests.TestImage) //nolint:gosec
	if err := cmd.Run(); err != nil {
		t.Fatalf(err.Error())
	}

	// check if the expected image actually exists
	result, err := podman.ImageExists(tests.TestImage)
	assert.Nil(t, err)
	assert.Equals(t, *result, true)

	// check if the expected image actually exists
	result, err = podman.ImageExists(tests.TestImageNoTag)
	assert.Nil(t, err)
	assert.Equals(t, *result, true)

	// check if same image but with different tag exists
	result, err = podman.ImageExists(tests.TestNotExistingTag)
	assert.Nil(t, err)
	assert.Equals(t, *result, false)

	// check if a not existing image exists
	result, err = podman.ImageExists(tests.TestNotExistingImage)
	assert.Nil(t, err)
	assert.Equals(t, *result, false)

	// cleanup
	tests.RemoveImage(logger, tests.TestImage)
}

func TestPodman_PullImage(t *testing.T) {
	logger := log.NewTestLogger(t)
	tests.RemoveImage(logger, tests.TestImage)

	podman := NewCliWrapper(tests.GetPodmanPath(), logger)
	assert.NotNil(t, tests.GetPodmanPath())

	// pull without platform
	if err := podman.PullImage(tests.TestImage, nil); err != nil {
		assert.Nil(t, err)
	}

	imageArch := tests.InspectImage(logger, tests.TestImage)
	assert.NotNil(t, imageArch)

	tests.RemoveImage(logger, tests.TestImage)
	// pull with platform
	platform := "linux/arm64"
	if err := podman.PullImage(tests.TestImage, &platform); err != nil {
		assert.Nil(t, err)
	}
	imageArch = tests.InspectImage(logger, tests.TestImage)
	assert.Equals(t, platform, fmt.Sprintf("%s/%s", imageArch.Os, imageArch.Architecture))
	tests.RemoveImage(logger, tests.TestImage)

	// pull not existing image without baseUrl (cli interactively asks for the image repository)
	if err := podman.PullImage(tests.TestNotExistingImageNoBaseURL, nil); err != nil {
		assert.NotNil(t, err)
	}
}
