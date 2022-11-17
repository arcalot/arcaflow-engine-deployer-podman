package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"go.arcalot.io/assert"
	"os"
	"os/exec"
	"testing"
)

const testImage = "quay.io/podman/hello:latest"
const testImageWithoutTag = "quay.io/podman/hello"
const testNotExistingTag = "quay.io/podman/hello:v0"
const testNotExistingImage = "quay.io/podman/imatestidonotexist:latest"

type basicInspection struct {
	Architecture string `json:"Architecture"`
	Os           string `json:"Os"`
}

func getPodmanPath() string {
	if err := godotenv.Load("../env/test.env"); err != nil {
		panic(err)
	}
	return os.Getenv("PODMAN_PATH")
}

func removeImage(image string) {
	cmd := exec.Command(getPodmanPath(), "rmi", "-f", image)
	cmd.Run()
}

func inspectImage(image string) *basicInspection {
	cmd := exec.Command(getPodmanPath(), "inspect", image)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Run()
	var objects []basicInspection
	if err := json.Unmarshal(out.Bytes(), &objects); err != nil {
		panic(err)
	}
	if len(objects) == 0 {
		return nil
	}
	return &objects[0]
}

func TestPodman_ImageExists(t *testing.T) {

	removeImage(testImage)

	podman := NewPodman(getPodmanPath())
	assert.NotNil(t, getPodmanPath())

	cmd := exec.Command(getPodmanPath(), "pull", testImage)
	if err := cmd.Run(); err != nil {
		t.Fatalf(err.Error())
	}

	// check if the expected image actually exists
	result, err := podman.ImageExists(testImage)
	assert.Nil(t, err)
	assert.Equals(t, *result, true)

	// check if the expected image actually exists
	result, err = podman.ImageExists(testImageWithoutTag)
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
	removeImage(testImage)

}

func TestPodman_PullImage(t *testing.T) {
	removeImage(testImage)

	podman := NewPodman(getPodmanPath())
	assert.NotNil(t, getPodmanPath())

	// pull without platform
	if err := podman.PullImage(testImage, nil); err != nil {
		assert.Nil(t, err)
	}

	imageArch := inspectImage(testImage)
	assert.NotNil(t, imageArch)

	removeImage(testImage)
	// pull with platform
	platform := "linux/arm64"
	if err := podman.PullImage(testImage, &platform); err != nil {
		assert.Nil(t, err)
	}
	imageArch = inspectImage(testImage)
	assert.Equals(t, platform, fmt.Sprintf("%s/%s", imageArch.Os, imageArch.Architecture))

	//TODO: FINISH TESTS
}
