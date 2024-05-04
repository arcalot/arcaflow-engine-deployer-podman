package cliwrapper

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"testing"

	log "go.arcalot.io/log/v2"

	"go.arcalot.io/assert"
	"go.flow.arcalot.io/podmandeployer/tests"
)

func podmanImageExists(t *testing.T, connectionName *string) {
	logger := log.NewTestLogger(t)
	tests.RemoveImage(logger, tests.TestImage)

	podman := NewCliWrapper(tests.GetPodmanPath(), logger, connectionName)

	assert.NotNil(t, tests.GetPodmanPath())

	cmd := exec.Command(tests.GetPodmanPath(), "pull", tests.TestImage) //nolint:gosec  // Command line is trusted
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
	podmanImageExists(t, nil)
}

func TestPodman_Remote_ImageExists(t *testing.T) {
	// Check if there is an existing connection of `podman-machine-default`
	// since this is included when installing podman desktop for macOS.
	connectionName := "podman-machine-default"
	chkDefaultConnectionCmd := exec.Command(tests.GetPodmanPath(), "--connection", connectionName, "system", "info") //nolint:gosec  // Command line is trusted
	if err := chkDefaultConnectionCmd.Run(); err != nil {
		// The podman-machine-default connection doesn't exist, so try to create
		// an alternative connection service.  For now, only try this on Linux.
		//
		//goland:noinspection GoBoolExpressions  // The linter cannot tell that this expression is not constant.
		if runtime.GOOS != "linux" {
			t.Skipf("There is no default Podman connection and no support for creating it on %s.", runtime.GOOS)
		}

		connectionName = createPodmanConnection(t)
	}

	// Run the test
	podmanImageExists(t, &connectionName)
}

// createPodmanConnection creates a Podman API service process and configures
// a Podman "connection" to allow it to be used for remote Podman invocations.
func createPodmanConnection(t *testing.T) (connectionName string) {
	// Setup:  create a temporary directory with a random name, to avoid
	// collisions with other concurrently-running tests; use the resulting
	// path as the name of the Podman service connection and put the service
	// socket in the directory.  Start a listener on that socket and configure
	// a connection to it.  Declare cleanup functions which will remove the
	// connection, kill the listener, and remove the temporary directory and
	// socket.
	t.Logf("Adding a local Podman API service and connection.")
	sockDir, err := os.MkdirTemp("", "arcaflow-engine-deployer-podman-test-*")
	if err != nil {
		t.Fatalf("Unable to create socket directory: %q", err)
	}

	t.Cleanup(func() {
		t.Logf("Removing socket directory, %q.", sockDir)
		if err := os.RemoveAll(sockDir); err != nil {
			t.Logf("Unable to remove socket directory, %q: %q", sockDir, err)
		}
	})

	t.Logf("Local Podman API service connection is %q.", sockDir)

	connectionName = sockDir
	podmanSocketPath := "unix://" + sockDir + "/podman.sock"

	podmanApiServiceCmd := exec.Command(tests.GetPodmanPath(), "system", "service", "--time=0", podmanSocketPath) //nolint:gosec  // Command line is trusted
	if err := podmanApiServiceCmd.Start(); err != nil {
		t.Fatal("Failed to create temporary Podman API service process")
	}

	t.Cleanup(func() {
		t.Logf("Killing the Podman API service process.")
		if err := podmanApiServiceCmd.Process.Kill(); err != nil {
			t.Fatal("Failed to kill Podman API service process.")
		}
	})

	addConnectionCmd := exec.Command(tests.GetPodmanPath(), "system", "connection", "add", connectionName, podmanSocketPath) //nolint:gosec  // Command line is trusted
	if err := addConnectionCmd.Run(); err != nil {
		t.Fatalf("Failed to add connection %q.", connectionName)
	}

	t.Cleanup(func() {
		t.Logf("Removing the Podman connection.")
		delConnectionCmd := exec.Command(tests.GetPodmanPath(), "system", "connection", "remove", connectionName) //nolint:gosec  // Command line is trusted
		if err := delConnectionCmd.Run(); err != nil {
			t.Fatalf("Failed to delete connection %q.", connectionName)
		}
	})

	return connectionName
}

func TestPodman_PullImage(t *testing.T) {
	logger := log.NewTestLogger(t)
	tests.RemoveImage(logger, tests.TestImageMultiPlatform)

	podman := NewCliWrapper(tests.GetPodmanPath(), logger, nil)
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
