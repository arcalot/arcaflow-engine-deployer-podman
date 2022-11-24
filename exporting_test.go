package podman

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
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
	if err := godotenv.Load("env/test.env"); err != nil {
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

// getHostCgroupNs detects the user's cgroup namespace
func getHostCgroupNs() string {
	var pid int = 0
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		cmd1 := exec.Command("/usr/bin/sleep", "3")
		cmd1.Start()
		pid = cmd1.Process.Pid
		cmd1.Wait()
	}()
	time.Sleep(1 * time.Second)
	wg.Add(1)
	var userCgroupNs string
	go func() {
		defer wg.Done()
		var stdout bytes.Buffer
		cmd2 := exec.Command("ls", "-al", fmt.Sprintf("/proc/%d/ns/cgroup", pid))
		cmd2.Stdout = &stdout
		cmd2.Run()
		userCgroupNs = strings.Split(stdout.String(), " ")[10]
	}()
	wg.Wait()
	//removes linux cgroup notation
	regex := regexp.MustCompile("cgroup\\:\\[(\\d+)\\]")
	userCgroupNs = regex.ReplaceAllString(userCgroupNs, "$1")
	userCgroupNs = strings.TrimSuffix(userCgroupNs, "\n")
	return userCgroupNs
}

// getPomanCgroupNs  detects the running container cgroup namespace
func getPomanCgroupNs(podmanPath string, containerName string) string {
	var wg sync.WaitGroup
	wg.Add(1)
	var podmanCgroupNs string
	go func() {
		defer wg.Done()
		var stdout bytes.Buffer
		cmd := exec.Command(podmanPath, "ps", "--ns", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.CGROUPNS}}")
		cmd.Stdout = &stdout
		cmd.Run()
		podmanCgroupNs = stdout.String()
	}()
	wg.Wait()
	podmanCgroupNs = strings.TrimSuffix(podmanCgroupNs, "\n")
	return podmanCgroupNs
}
