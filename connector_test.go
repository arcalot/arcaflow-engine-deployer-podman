package podman

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go.arcalot.io/assert"
	"go.arcalot.io/log"
	"go.flow.arcalot.io/deployer"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

func getTestRandomString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

func getConnector(t *testing.T, configJson string) (deployer.Connector, *Config) {

	var config any
	if err := json.Unmarshal([]byte(configJson), &config); err != nil {
		t.Fatal(err)
	}

	factory := NewFactory()
	schema := factory.ConfigurationSchema()
	unserializedConfig, err := schema.UnserializeType(config)
	assert.NoError(t, err)
	connector, err := factory.Create(unserializedConfig, log.NewTestLogger(t))
	assert.NoError(t, err)
	return connector, unserializedConfig
}

var inOutConfig = `
{
   "podman":{
      "path":"/usr/bin/podman"
   }
}
`

func TestSimpleInOut(t *testing.T) {
	connector, _ := getConnector(t, inOutConfig)
	container, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, container.Close())
	})

	var containerInput = []byte("input abc\n")
	assert.NoErrorR[int](t)(container.Write(containerInput))

	buf := new(strings.Builder)
	assert.NoErrorR[int64](t)(io.Copy(buf, container))
	assert.Contains(t, buf.String(), "This is what input was received: \"abc\"")
}

var envConfig = `
{
   "deployment":{
      "container":{
         "NetworkDisabled":true,
         "Env":[
            "DEPLOYER_PODMAN_TEST_1=TEST1",
            "DEPLOYER_PODMAN_TEST_2=TEST2"
         ]
      }
   },
   "podman":{
      "path":"/usr/bin/podman"
   }
}
`

func TestEnv(t *testing.T) {
	envVar1 := "DEPLOYER_PODMAN_TEST_1=TEST1"
	envVar2 := "DEPLOYER_PODMAN_TEST_2=TEST2"
	connector, _ := getConnector(t, envConfig)
	container, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, container.Close())
	})

	var containerInput = []byte("env\n")

	assert.NoErrorR[int](t)(container.Write(containerInput))

	buf := new(strings.Builder)
	assert.NoErrorR[int64](t)(io.Copy(buf, container))
	assert.Contains(t, buf.String(), envVar1)
	assert.Contains(t, buf.String(), envVar2)
}

var volumeConfig = `
{
   "deployment":{
      "host":{
         "Binds":[
            "./test/volumes:/test"
         ]
      }
   },
   "podman":{
      "path":"/usr/bin/podman"
   }
}
`

func TestSimpleVolume(t *testing.T) {
	fileContent, err := os.ReadFile("./test/volumes/test_file.txt")
	assert.Nil(t, err)
	connector, _ := getConnector(t, volumeConfig)

	cwd, err := os.Getwd()
	assert.Nil(t, err)
	//disable selinux on the test folder in order to make the file readable from within the container
	cmd := exec.Command("chcon", "-Rt", "svirt_sandbox_file_t", fmt.Sprintf("%s/test/volumes", cwd))
	err = cmd.Run()
	assert.Nil(t, err)

	container, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, container.Close())
	})

	var containerInput = []byte("volume\n")

	assert.NoErrorR[int](t)(container.Write(containerInput))

	buf := new(strings.Builder)
	assert.NoErrorR[int64](t)(io.Copy(buf, container))
	assert.Contains(t, buf.String(), string(fileContent))
}

var nameTemplate = `
{
   "podman":{
      "path":"/usr/bin/podman",
      "containerName":"%s"
   }
}
`

func TestContainerName(t *testing.T) {
	containerName := fmt.Sprintf("test_%s", getTestRandomString(5))
	configTemplate := fmt.Sprintf(nameTemplate, containerName)
	connector, config := getConnector(t, configTemplate)

	container, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)

	t.Cleanup(func() {
		cmd := exec.Command(config.Podman.Path, "container", "rm", containerName)
		cmd.Run()
		assert.NoError(t, container.Close())
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var containerInput = []byte("sleep 3\n")
		assert.NoErrorR[int](t)(container.Write(containerInput))
	}()

	var stdout bytes.Buffer
	cmd := exec.Command(config.Podman.Path, "ps", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.ID}}")
	cmd.Stdout = &stdout
	cmd.Run()
	stdoutStr := stdout.String()
	assert.NotNil(t, stdoutStr)
	wg.Wait()

}

var cgroupTemplate = `
{
   "podman":{
      "path":"/usr/bin/podman",
      "containerName":"%s"
      "cgroupNs":"%s"
   }
}
`

func TestContainerCgroupNs(t *testing.T) {
	containername1 := fmt.Sprintf("test_%s", getTestRandomString(5))
	configtemplate1 := fmt.Sprintf(cgroupTemplate, containername1, "private")
	connector1, config := getConnector(t, configtemplate1)

	container1, err := connector1.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)

	containername2 := fmt.Sprintf("test_%s", getTestRandomString(5))
	configtemplate2 := fmt.Sprintf(cgroupTemplate, containername2, fmt.Sprintf("container:%s", containername1))
	connector2, _ := getConnector(t, configtemplate2)

	container2, err := connector2.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)

	t.Cleanup(func() {
		cmd := exec.Command(config.Podman.Path, "container", "rm", containername1)
		cmd.Run()
		cmd = exec.Command(config.Podman.Path, "container", "rm", containername2)
		cmd.Run()
		assert.NoError(t, container1.Close())
		assert.NoError(t, container2.Close())
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var containerInput = []byte("sleep 10\n")
		assert.NoErrorR[int](t)(container1.Write(containerInput))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var containerInput = []byte("sleep 10\n")
		assert.NoErrorR[int](t)(container1.Write(containerInput))
	}()
	wg.Wait()

}

// /usr/bin/readlink /proc/*/task/*/ns/* | sort -u | grep cgroup
// podman ps --ns --format {{.CGROUPNS}}
// podman run --cgroupns private --name -i -d docker.io/library/bash:latest /bin/sleep 3000
// podman ps --filter name=test_1
// podman ps --filter name=test_1 --format {{.ID}}
// podman delete test_1
func TestPrivateCgroupNs(t *testing.T) {
	/*connector, config := getConnector(t, fmt.Sprintf(cgroupTemplate, "private"))
	container, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")

	cmd := exec.Command(config.Podman.Path, "ps", "--ns", "--format", "{{.CGROUPNS}}")
	err := cmd.Run()*/
}

func TestHostCgroupNs(t *testing.T) {

}
