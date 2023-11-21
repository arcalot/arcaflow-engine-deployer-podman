package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "go.arcalot.io/log/v2"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

const TestImage = "quay.io/podman/hello:latest"
const TestImageNoTag = "quay.io/podman/hello"
const TestImageNoBaseURL = "hello:latest"
const TestNotExistingTag = "quay.io/podman/hello:v0"
const TestNotExistingImage = "quay.io/podman/imatestidonotexist:latest"
const TestNotExistingImageNoBaseURL = "imatestidonotexist:latest"

type BasicInspection struct {
	Architecture string `json:"Architecture"`
	Os           string `json:"Os"`
}

func GetPodmanPath() string {
	envPath := os.Getenv("PODMAN_PATH")
	if len(envPath) > 0 {
		return envPath
	}
	return "podman"
}

func RemoveImage(logger log.Logger, image string) {
	cmd := exec.Command(GetPodmanPath(), "rmi", "-f", image) //nolint:gosec
	if err := cmd.Run(); err != nil {
		logger.Errorf("failed to remove image %s", image)
	}
}

func InspectImage(logger log.Logger, image string) *BasicInspection {
	cmd := exec.Command(GetPodmanPath(), "inspect", image) //nolint:gosec
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		logger.Errorf(err.Error())
	}
	var objects []BasicInspection
	if err := json.Unmarshal(out.Bytes(), &objects); err != nil {
		logger.Errorf(err.Error())
	}
	if len(objects) == 0 {
		return nil
	}
	return &objects[0]
}

// GetCommmandCgroupNs detects the user's cgroup namespace
func GetCommmandCgroupNs(logger log.Logger, command string, args []string) string {
	// determine pid of a process executed by this user
	var pid int
	cmd1 := exec.Command(command, args...)
	if err := cmd1.Start(); err != nil {
		logger.Errorf(err.Error())
	}
	pid = cmd1.Process.Pid

	// wait
	time.Sleep(1 * time.Second)

	// determine the cgroup using a pid
	var userCgroupNs string
	var stdout bytes.Buffer
	// execute a shell-like command to list the cgroups in the namespace of the pid
	cmd2 := exec.Command("ls", "-al", fmt.Sprintf("/proc/%d/ns/cgroup", pid)) //nolint:gosec
	cmd2.Stdout = &stdout
	if err := cmd2.Run(); err != nil {
		logger.Errorf(err.Error())
	}
	// parse output from command
	stdoutStr := stdout.String()
	regex := regexp.MustCompile(`.*cgroup:\[(\d+)\]`)
	userCgroupNs = regex.ReplaceAllString(stdoutStr, "$1")
	userCgroupNs = strings.TrimSuffix(userCgroupNs, "\n")

	if err := cmd1.Wait(); err != nil {
		logger.Errorf(err.Error())
	}

	return userCgroupNs
}

// GetPodmanCgroupNs  detects the running container cgroup namespace
func GetPodmanCgroupNs(logger log.Logger, podmanPath string, containerName string) string {
	var wg sync.WaitGroup
	wg.Add(1)
	var podmanCgroupNs string
	go func() {
		defer wg.Done()
		var stdout bytes.Buffer
		cmd := exec.Command(podmanPath, "ps", "--ns", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.CGROUPNS}}") //nolint:gosec
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			logger.Errorf(err.Error())
		}
		podmanCgroupNs = stdout.String()
	}()
	wg.Wait()
	podmanCgroupNs = strings.TrimSuffix(podmanCgroupNs, "\n")
	return podmanCgroupNs
}

func IsContainerRunning(logger log.Logger, podmanPath string, containerName string) bool {
	var stdout bytes.Buffer
	cmd := exec.Command(podmanPath, "ps", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.ID}}") //nolint:gosec
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		logger.Errorf(err.Error())
	}
	stdoutStr := stdout.String()
	return stdoutStr != ""
}

func GetPodmanPsNsWithFormat(logger log.Logger, podmanPath string, containerName string, format string) string {
	var stdoutContainer bytes.Buffer
	cmd := exec.Command(podmanPath, "ps", "--ns", "--filter", fmt.Sprintf("name=%s", containerName), "--format", format) //nolint:gosec
	cmd.Stdout = &stdoutContainer
	if err := cmd.Run(); err != nil {
		logger.Errorf(err.Error())
	}
	return strings.TrimSuffix(stdoutContainer.String(), "\n")
}

func IsRunningOnGithub() bool {
	githubEnv := os.Getenv("GITHUB_ACTION")
	return githubEnv != ""
}
