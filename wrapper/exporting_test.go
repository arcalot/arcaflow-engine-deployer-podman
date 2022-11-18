package wrapper

import (
	"bytes"
	"encoding/json"
	"github.com/joho/godotenv"
	"os"
	"os/exec"
)

const testImage = "quay.io/podman/hello:latest:latest"
const testImageIo = "registry.fedoraproject.org/fedora"
const testImageNoTag = "quay.io/podman/hello"
const testImageNoBaseUrl = "hello:latest"
const testNotExistingTag = "quay.io/podman/hello:v0"
const testNotExistingImage = "quay.io/podman/imatestidonotexist:latest"
const testNotExistingImageNoBaseUrl = "imatestidonotexist:latest"

type basicInspection struct {
	Architecture string `json:"Architecture"`
	Os           string `json:"Os"`
}

func GetPodmanPath() string {
	if err := godotenv.Load("../env/test.env"); err != nil {
		panic(err)
	}
	return os.Getenv("PODMAN_PATH")
}

func RemoveImage(image string) {
	cmd := exec.Command(GetPodmanPath(), "rmi", "-f", image)
	cmd.Run()
}

func InspectImage(image string) *basicInspection {
	cmd := exec.Command(GetPodmanPath(), "inspect", image)
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
