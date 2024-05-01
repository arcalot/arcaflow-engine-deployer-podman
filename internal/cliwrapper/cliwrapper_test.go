package cliwrapper

import (
	"fmt"
	"os/exec"
	"runtime"
	"testing"

	log "go.arcalot.io/log/v2"

	"go.arcalot.io/assert"
	"go.flow.arcalot.io/podmandeployer/tests"
)

func Podman_ImageExists(t *testing.T, connectionName string) {
	logger := log.NewTestLogger(t)
	tests.RemoveImage(logger, tests.TestImage)

	podman := NewCliWrapper(tests.GetPodmanPath(), logger, connectionName)

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

func TestPodman_ImageExists(t *testing.T) {
	Podman_ImageExists(t, "")
}

func TestPodman_Remote_ImageExists(t *testing.T) {
	var tmpPodmanSocketCmd *exec.Cmd

	// Check if there is an existing connection of `podman-machine-default` since this is included when installing
	// podman desktop for macOS.
	connectionName := "podman-machine-default"
	chkDefaultConnectionCmd := exec.Command(tests.GetPodmanPath(), "--connection", connectionName, "system", "info") //nolint:gosec
	if err := chkDefaultConnectionCmd.Run(); err == nil {
		Podman_ImageExists(t, connectionName)
	} else if runtime.GOOS == "linux" {
		// The podman-machine-default doesn't exist then for Linux, create a temporary socket

		// Setup
		connectionName = "arcaflow-engine-deployer-podman-test"
		podmanSocketPath := "unix:///var/tmp/" + connectionName + ".sock"

		tmpPodmanSocketCmd = exec.Command(tests.GetPodmanPath(), "system", "service", "--time=0", podmanSocketPath)
		if err := tmpPodmanSocketCmd.Start(); err != nil {
			t.Fatalf("Failed to create temporary podman socket")
		}

		addConnectionCmd := exec.Command(tests.GetPodmanPath(), "system", "connection", "add", connectionName, podmanSocketPath)
		if err := addConnectionCmd.Run(); err != nil {
			t.Fatalf("Failed to add connection: " + connectionName)
		}

		// Run test
		Podman_ImageExists(t, connectionName)

		// Clean up
		if err := tmpPodmanSocketCmd.Process.Kill(); err != nil {
			t.Fatalf("Failed to kill temporary socket")
		}

		delConnectionCmd := exec.Command(tests.GetPodmanPath(), "system", "connection", "remove", connectionName)
		if err := delConnectionCmd.Run(); err != nil {
			t.Fatalf("Failed to delete connection: " + connectionName)
		}
		// Unexpected setup, force user to add podman-machine-default
	} else {
		t.Fatalf("Unsupported configuration")
	}
}

func TestPodman_PullImage(t *testing.T) {
	logger := log.NewTestLogger(t)
	tests.RemoveImage(logger, tests.TestImageMultiPlatform)

	podman := NewCliWrapper(tests.GetPodmanPath(), logger, "")
	assert.NotNil(t, tests.GetPodmanPath())

	// pull without platform
	if err := podman.PullImage(tests.TestImageMultiPlatform, nil); err != nil {
		assert.Nil(t, err)
	}

	imageArch := tests.InspectImage(logger, tests.TestImageMultiPlatform)
	assert.NotNil(t, imageArch)

	tests.RemoveImage(logger, tests.TestImageMultiPlatform)
	// pull with platform
	platform := "linux/arm64"
	if err := podman.PullImage(tests.TestImageMultiPlatform, &platform); err != nil {
		assert.Nil(t, err)
	}
	imageArch = tests.InspectImage(logger, tests.TestImageMultiPlatform)
	assert.Equals(t, platform, fmt.Sprintf("%s/%s", imageArch.Os, imageArch.Architecture))
	tests.RemoveImage(logger, tests.TestImageMultiPlatform)

	// pull not existing image without baseUrl (cli interactively asks for the image repository)
	if err := podman.PullImage(tests.TestNotExistingImageNoBaseURL, nil); err != nil {
		assert.NotNil(t, err)
	}
}
