package podman

import (
	"fmt"
	"go.arcalot.io/assert"
	"os/exec"
	"testing"
)

func TestPodman_ImageExists(t *testing.T) {

	RemoveImage(testImage)

	podman := NewCliWrapper(GetPodmanPath())
	assert.NotNil(t, GetPodmanPath())

	cmd := exec.Command(GetPodmanPath(), "pull", testImage)
	if err := cmd.Run(); err != nil {
		t.Fatalf(err.Error())
	}

	// check if the expected image actually exists
	result, err := podman.ImageExists(testImage)
	assert.Nil(t, err)
	assert.Equals(t, *result, true)

	// check if the expected image actually exists
	result, err = podman.ImageExists(testImageNoTag)
	assert.Nil(t, err)
	assert.Equals(t, *result, true)

	// check if same image but with different tag exists
	result, err = podman.ImageExists(testNotExistingTag)
	assert.Nil(t, err)
	assert.Equals(t, *result, false)

	// check if a not existing image exists
	result, err = podman.ImageExists(testNotExistingImage)
	assert.Nil(t, err)
	assert.Equals(t, *result, false)

	//cleanup
	RemoveImage(testImage)

}

func TestPodman_PullImage(t *testing.T) {

	RemoveImage(testImage)

	podman := NewCliWrapper(GetPodmanPath())
	assert.NotNil(t, GetPodmanPath())

	// pull without platform
	if err := podman.PullImage(testImage, nil); err != nil {
		assert.Nil(t, err)
	}

	imageArch := InspectImage(testImage)
	assert.NotNil(t, imageArch)

	RemoveImage(testImage)
	// pull with platform
	platform := "linux/arm64"
	if err := podman.PullImage(testImage, &platform); err != nil {
		assert.Nil(t, err)
	}
	imageArch = InspectImage(testImage)
	assert.Equals(t, platform, fmt.Sprintf("%s/%s", imageArch.Os, imageArch.Architecture))

	RemoveImage(testImage)
	// pull existing image without baseUrl
	if err := podman.PullImage(testImageNoBaseUrl, nil); err != nil {
		assert.Nil(t, err)
	}
	imageArch = InspectImage(testImage)
	assert.NotNil(t, imageArch)

	//pull not existing image without baseUrl (cli interactively asks for the image repository)
	if err := podman.PullImage(testNotExistingImageNoBaseUrl, nil); err != nil {
		assert.NotNil(t, err)
	}

}
