package cli_wrapper

import (
	podman2 "arcaflow-engine-deployer-podman"
	"fmt"
	"go.arcalot.io/assert"
	"os/exec"
	"testing"
)

func TestPodman_ImageExists(t *testing.T) {

	podman2.RemoveImage(podman2.testImage)

	podman := NewCliWrapper(podman2.GetPodmanPath())
	assert.NotNil(t, podman2.GetPodmanPath())

	cmd := exec.Command(podman2.GetPodmanPath(), "pull", podman2.testImage)
	if err := cmd.Run(); err != nil {
		t.Fatalf(err.Error())
	}

	// check if the expected image actually exists
	result, err := podman.ImageExists(podman2.testImage)
	assert.Nil(t, err)
	assert.Equals(t, *result, true)

	// check if the expected image actually exists
	result, err = podman.ImageExists(podman2.testImageNoTag)
	assert.Nil(t, err)
	assert.Equals(t, *result, true)

	// check if same image but with different tag exists
	result, err = podman.ImageExists(podman2.testNotExistingTag)
	assert.Nil(t, err)
	assert.Equals(t, *result, false)

	// check if a not existing image exists
	result, err = podman.ImageExists(podman2.testNotExistingImage)
	assert.Nil(t, err)
	assert.Equals(t, *result, false)

	//cleanup
	podman2.RemoveImage(podman2.testImage)

}

func TestPodman_PullImage(t *testing.T) {

	podman2.RemoveImage(podman2.testImage)

	podman := NewCliWrapper(podman2.GetPodmanPath())
	assert.NotNil(t, podman2.GetPodmanPath())

	// pull without platform
	if err := podman.PullImage(podman2.testImage, nil); err != nil {
		assert.Nil(t, err)
	}

	imageArch := podman2.InspectImage(podman2.testImage)
	assert.NotNil(t, imageArch)

	podman2.RemoveImage(podman2.testImage)
	// pull with platform
	platform := "linux/arm64"
	if err := podman.PullImage(podman2.testImage, &platform); err != nil {
		assert.Nil(t, err)
	}
	imageArch = podman2.InspectImage(podman2.testImage)
	assert.Equals(t, platform, fmt.Sprintf("%s/%s", imageArch.Os, imageArch.Architecture))

	podman2.RemoveImage(podman2.testImage)
	// pull existing image without baseUrl
	if err := podman.PullImage(podman2.testImageNoBaseUrl, nil); err != nil {
		assert.Nil(t, err)
	}
	imageArch = podman2.InspectImage(podman2.testImage)
	assert.NotNil(t, imageArch)

	//pull not existing image without baseUrl (cli interactively asks for the image repository)
	if err := podman.PullImage(podman2.testNotExistingImageNoBaseUrl, nil); err != nil {
		assert.NotNil(t, err)
	}

}
