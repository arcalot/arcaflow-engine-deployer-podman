package cli_wrapper

import (
    "fmt"
    "os/exec"
    "testing"

    "arcaflow-engine-deployer-podman/tests"
    "go.arcalot.io/assert"
)

func TestPodman_ImageExists(t *testing.T) {

    tests.RemoveImage(tests.TestImage)

    podman := NewCliWrapper(tests.GetPodmanPath())
    assert.NotNil(t, tests.GetPodmanPath())

    cmd := exec.Command(tests.GetPodmanPath(), "pull", tests.TestImage)
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
    tests.RemoveImage(tests.TestImage)

}

func TestPodman_PullImage(t *testing.T) {

    tests.RemoveImage(tests.TestImage)

    podman := NewCliWrapper(tests.GetPodmanPath())
    assert.NotNil(t, tests.GetPodmanPath())

    // pull without platform
    if err := podman.PullImage(tests.TestImage, nil); err != nil {
        assert.Nil(t, err)
    }

    imageArch := tests.InspectImage(tests.TestImage)
    assert.NotNil(t, imageArch)

    tests.RemoveImage(tests.TestImage)
    // pull with platform
    platform := "linux/arm64"
    if err := podman.PullImage(tests.TestImage, &platform); err != nil {
        assert.Nil(t, err)
    }
    imageArch = tests.InspectImage(tests.TestImage)
    assert.Equals(t, platform, fmt.Sprintf("%s/%s", imageArch.Os, imageArch.Architecture))

    tests.RemoveImage(tests.TestImage)
    // pull existing image without baseUrl
    if err := podman.PullImage(tests.TestImageNoBaseUrl, nil); err != nil {
        assert.Nil(t, err)
    }
    imageArch = tests.InspectImage(tests.TestImage)
    assert.NotNil(t, imageArch)

    // pull not existing image without baseUrl (cli interactively asks for the image repository)
    if err := podman.PullImage(tests.TestNotExistingImageNoBaseUrl, nil); err != nil {
        assert.NotNil(t, err)
    }

}
